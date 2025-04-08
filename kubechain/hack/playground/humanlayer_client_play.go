package main

import (
	"context"
	"flag"
	"fmt"
	"os"

	"github.com/google/uuid"
	kubechainv1alpha1 "github.com/humanlayer/smallchain/kubechain/api/v1alpha1"

	"github.com/humanlayer/smallchain/kubechain/internal/humanlayer"
	"github.com/humanlayer/smallchain/kubechain/internal/humanlayerapi"
)

func requestApproval(
	client humanlayer.HumanLayerClientWrapper,
	channelType kubechainv1alpha1.ContactChannelType,
) *humanlayerapi.FunctionCallOutput {
	switch channelType {
	case kubechainv1alpha1.ContactChannelTypeSlack:
		client.SetSlackConfig(&kubechainv1alpha1.SlackChannelConfig{
			ChannelOrUserID:           "C07HR5JL15F",
			ContextAboutChannelOrUser: "Channel for approving web fetch operations",
		})
	case kubechainv1alpha1.ContactChannelTypeEmail:
		client.SetEmailConfig(&kubechainv1alpha1.EmailChannelConfig{
			Address:          os.Getenv("HL_EXAMPLE_CONTACT_EMAIL"),
			ContextAboutUser: "Primary approver for web fetch operations",
		})
	default:
		panic("Unsupported channel type: " + channelType)
	}

	client.SetFunctionCallSpec("test-city", map[string]any{
		"a": 1,
		"b": 2,
	})

	client.SetCallID("call-" + uuid.New().String())
	client.SetRunID("sundeep-is-testing")

	functionCall, statusCode, err := client.RequestApproval(context.Background())

	fmt.Println(functionCall.GetCallId())
	fmt.Println(statusCode)
	fmt.Println(err)

	return functionCall
}

func requestHumanContact(
	client humanlayer.HumanLayerClientWrapper,
	channelType kubechainv1alpha1.ContactChannelType,
) *humanlayerapi.HumanContactOutput {
	switch channelType {
	case kubechainv1alpha1.ContactChannelTypeSlack:
		client.SetSlackConfig(&kubechainv1alpha1.SlackChannelConfig{
			ChannelOrUserID:           "C07HR5JL15F",
			ContextAboutChannelOrUser: "Channel for human contact",
		})
	case kubechainv1alpha1.ContactChannelTypeEmail:
		client.SetEmailConfig(&kubechainv1alpha1.EmailChannelConfig{
			Address:          os.Getenv("HL_EXAMPLE_CONTACT_EMAIL"),
			ContextAboutUser: "Contact for human interaction",
		})
	default:
		panic("Unsupported channel type: " + channelType)
	}

	client.SetCallID("call-" + uuid.New().String())
	client.SetRunID("sundeep-is-testing")

	humanContact, statusCode, err := client.RequestHumanContact(context.Background())

	fmt.Println(humanContact.GetCallId())
	fmt.Println(statusCode)
	fmt.Println(err)

	return humanContact
}

func getFunctionCallStatus(client humanlayer.HumanLayerClientWrapper) *humanlayerapi.FunctionCallOutput {
	functionCall, statusCode, err := client.GetFunctionCallStatus(context.Background())

	fmt.Println(functionCall.GetCallId())
	fmt.Println(statusCode)
	fmt.Println(err)

	return functionCall
}

func main() {
	// Define command line flags
	callIDFlag := flag.String("call-id", "", "Existing call ID to check status for")
	typeFlag := flag.String("channel", "slack", "Channel type (slack or email)")
	humanContactFlag := flag.Bool("human-contact", false, "Use human contact instead of approval")
	flag.Parse()

	factory, _ := humanlayer.NewHumanLayerClientFactory("")

	client := factory.NewHumanLayerClient()
	client.SetAPIKey(os.Getenv("HUMANLAYER_API_KEY"))

	var callID string

	if *callIDFlag != "" {
		fmt.Println("Call ID provided as argument - skipping request")
		callID = *callIDFlag
	} else if *humanContactFlag {
		hc := requestHumanContact(client, kubechainv1alpha1.ContactChannelType(*typeFlag))
		callID = hc.GetCallId()
	} else {
		fc := requestApproval(client, kubechainv1alpha1.ContactChannelType(*typeFlag))
		callID = fc.GetCallId()
	}

	client.SetCallID(callID)

	if *humanContactFlag {
		hc, _, _ := client.GetHumanContactStatus(context.Background())
		status := hc.GetStatus()

		response, ok := status.GetResponseOk()

		if !ok {
			fmt.Println("Not responded yet")
		} else {
			fmt.Println(*response)
		}
	} else {

		fcStatus := getFunctionCallStatus(client)
		status := fcStatus.GetStatus()

		approved, ok := status.GetApprovedOk()

		// Check if the value was set
		switch {
		case !ok:
			fmt.Println("Not responded yet")
		case approved == nil:
			fmt.Println("Approval status is nil (Not responded yet)")
		case *approved:
			fmt.Println("Approved")
		default:
			fmt.Println("Rejected")
		}

	}
}
