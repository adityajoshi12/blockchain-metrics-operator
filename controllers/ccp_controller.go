/*
Copyright 2022.

Licensed under the Apache License, Version 2.0 (the "License");
you may not use this file except in compliance with the License.
You may obtain a copy of the License at

    http://www.apache.org/licenses/LICENSE-2.0

Unless required by applicable law or agreed to in writing, software
distributed under the License is distributed on an "AS IS" BASIS,
WITHOUT WARRANTIES OR CONDITIONS OF ANY KIND, either express or implied.
See the License for the specific language governing permissions and
limitations under the License.
*/

package controllers

import (
	metricsv1 "blockchain.com/blockchain/api/v1"
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"github.com/hyperledger/fabric-protos-go/peer"
	"github.com/hyperledger/fabric-sdk-go/pkg/client/resmgmt"
	"github.com/hyperledger/fabric-sdk-go/pkg/common/providers/fab"
	"github.com/hyperledger/fabric-sdk-go/pkg/core/config"
	"github.com/hyperledger/fabric-sdk-go/pkg/fabsdk"
	"github.com/olekukonko/tablewriter"
	"github.com/prometheus/client_golang/prometheus"
	_ "github.com/prometheus/client_golang/prometheus"
	log "github.com/sirupsen/logrus"
	corev1 "k8s.io/api/core/v1"
	"k8s.io/apimachinery/pkg/api/errors"
	"k8s.io/apimachinery/pkg/runtime"
	"os"
	"os/exec"
	"reflect"
	ctrl "sigs.k8s.io/controller-runtime"
	"sigs.k8s.io/controller-runtime/pkg/client"
	"sigs.k8s.io/controller-runtime/pkg/metrics"
	"strings"
	"time"
)

var (
	blockchainHeight = prometheus.NewGaugeVec(prometheus.GaugeOpts{
		Namespace: "ledger",
		Subsystem: "",
		Name:      "blockchain_height",
		Help:      "Height of the chain in blocks.",
	},
		[]string{
			"msp_id",
			"peer_url",
			"channel",
		})
)

func init() {
	// Register custom metrics with the global prometheus registry
	metrics.Registry.Register(blockchainHeight)
}

// CCPReconciler reconciles a CCP object
type CCPReconciler struct {
	client.Client
	Scheme *runtime.Scheme
}

//+kubebuilder:rbac:groups=metrics.blockchain.com,resources=ccps,verbs=get;list;watch;create;update;patch;delete
//+kubebuilder:rbac:groups=metrics.blockchain.com,resources=ccps/status,verbs=get;update;patch
//+kubebuilder:rbac:groups=metrics.blockchain.com,resources=ccps/finalizers,verbs=update

func (r *CCPReconciler) Reconcile(ctx context.Context, req ctrl.Request) (ctrl.Result, error) {

	log.Info("Reconcile called")
	ccp := metricsv1.CCP{}
	if err := r.Get(ctx, req.NamespacedName, &ccp); err != nil {
		if errors.IsNotFound(err) {
			// object not found, could have been deleted after
			// reconcile request, hence don't requeue
			return ctrl.Result{}, nil
		}
		return ctrl.Result{}, err
	}

	mspID, peerId, user := getOrgDetails(ccp.Spec.ConnectionProfile)
	reqBodyBytes := new(bytes.Buffer)
	err := json.NewEncoder(reqBodyBytes).Encode(ccp.Spec.ConnectionProfile)
	if err != nil {
		return ctrl.Result{}, err
	}

	getOrgDetails(ccp.Spec.ConnectionProfile)

	configBackend := config.FromRaw(reqBodyBytes.Bytes(), "json")

	sdk, err := fabsdk.New(configBackend)
	if err != nil {
		log.Errorf("failed to create sdk")
		ccp.Status.Status = "Failed"
		ccp.Status.Message = "failed to create sdk"
		err := r.Update(ctx, &ccp)
		if err != nil {
			return ctrl.Result{RequeueAfter: time.Duration(ccp.Spec.SyncTime) * time.Second}, err
		}
		return ctrl.Result{RequeueAfter: time.Duration(ccp.Spec.SyncTime) * time.Second}, err
	}

	clientContext := sdk.Context(fabsdk.WithUser(user), fabsdk.WithOrg(mspID))

	resClient, err := resmgmt.New(clientContext)

	channels, err := resClient.QueryChannels(resmgmt.WithTargetEndpoints(peerId))
	if err != nil {
		log.Errorf("failed to query channels")
		ccp.Status.Status = "Failed"
		ccp.Status.Message = "failed to query channels"
		err := r.Update(ctx, &ccp)
		if err != nil {
			return ctrl.Result{RequeueAfter: time.Duration(ccp.Spec.SyncTime) * time.Second}, err
		}
		return ctrl.Result{RequeueAfter: time.Duration(ccp.Spec.SyncTime) * time.Second}, err
	}
	getBlocks(channels, sdk, user, mspID)

	ccp.Status.Status = "Running"
	ccp.Status.Message = "running"
	err = r.Update(context.TODO(), &ccp)
	if err != nil {
		return ctrl.Result{RequeueAfter: time.Duration(ccp.Spec.SyncTime) * time.Second}, err
	}
	return ctrl.Result{RequeueAfter: time.Duration(ccp.Spec.SyncTime) * time.Second}, nil
}

// SetupWithManager sets up the controller with the Manager.
func (r *CCPReconciler) SetupWithManager(mgr ctrl.Manager) error {
	return ctrl.NewControllerManagedBy(mgr).
		For(&metricsv1.CCP{}).
		Owns(&corev1.ConfigMap{}).
		Complete(r)
}

var floatType = reflect.TypeOf(float64(0))

func toFloat(obj interface{}) (float64, error) {
	v := reflect.ValueOf(obj)
	v = reflect.Indirect(v)
	if !v.Type().ConvertibleTo(floatType) {
		return 0, fmt.Errorf("cannot convert %v to float64", v.Type())
	}
	fv := v.Convert(floatType)
	return fv.Float(), nil
}

func renderTable(data [][]string) {
	tableString := &strings.Builder{}
	table := tablewriter.NewWriter(tableString)
	table.SetHeader([]string{"URL", "MSP ID", "Height"})
	table.SetAutoWrapText(false)
	table.SetAutoFormatHeaders(true)
	table.SetHeaderAlignment(tablewriter.ALIGN_LEFT)
	table.SetAlignment(tablewriter.ALIGN_LEFT)
	table.SetCenterSeparator("")
	table.SetColumnSeparator("")
	table.SetRowSeparator("")
	table.SetTablePadding("\t")
	table.SetNoWhiteSpace(true)
	table.AppendBulk(data) // Add Bulk Data
	table.Render()
	cmd := exec.Command("clear") //Linux example, its tested
	cmd.Stdout = os.Stdout
	err := cmd.Run()
	if err != nil {

	}
	fmt.Printf("\r%s", tableString.String())
}

func getBlocks(channels *peer.ChannelQueryResponse, sdk *fabsdk.FabricSDK, user, mspID string) {
	for _, channel := range channels.Channels {

		var data [][]string
		org1ChannelContext := sdk.ChannelContext(
			channel.GetChannelId(),
			fabsdk.WithUser(user),
			fabsdk.WithOrg(mspID),
		)
		chCtx, err := org1ChannelContext()
		if err != nil {
			log.Error(err)
		}
		discovery, err := chCtx.ChannelService().Discovery()
		if err != nil {
			log.Fatal(err)
		}
		peers, err := discovery.GetPeers()
		if err != nil {
			log.Printf("Failed to get peers %v", err)
		}

		for _, peer := range peers {
			props := peer.Properties()
			ledgerHeight := props[fab.PropertyLedgerHeight]
			data = append(data, []string{
				peer.URL(), peer.MSPID(), fmt.Sprintf("L=%d", ledgerHeight),
			})
			blocks, _ := toFloat(ledgerHeight)
			blockchainHeight.With(
				prometheus.Labels{
					"msp_id": peer.MSPID(), "peer_url": peer.URL(), "channel": channel.ChannelId,
				}).Set(blocks)
		}
		renderTable(data)

	}
}

func getOrgDetails(ccp metricsv1.ConnectionProfile) (mspID, peerId, userId string) {
	for _, det := range ccp.Organizations {
		mspID = det.Mspid
		peerId = det.Peers[0]
		for uId, _ := range det.Users {
			userId = uId
			break
		}
		break
	}
	return
}
