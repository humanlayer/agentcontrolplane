# Escalation

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**EscalationMsg** | **string** |  | 
**AdditionalRecipients** | Pointer to [**[]EmailRecipient**](EmailRecipient.md) |  | [optional] 

## Methods

### NewEscalation

`func NewEscalation(escalationMsg string, ) *Escalation`

NewEscalation instantiates a new Escalation object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewEscalationWithDefaults

`func NewEscalationWithDefaults() *Escalation`

NewEscalationWithDefaults instantiates a new Escalation object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetEscalationMsg

`func (o *Escalation) GetEscalationMsg() string`

GetEscalationMsg returns the EscalationMsg field if non-nil, zero value otherwise.

### GetEscalationMsgOk

`func (o *Escalation) GetEscalationMsgOk() (*string, bool)`

GetEscalationMsgOk returns a tuple with the EscalationMsg field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEscalationMsg

`func (o *Escalation) SetEscalationMsg(v string)`

SetEscalationMsg sets EscalationMsg field to given value.


### GetAdditionalRecipients

`func (o *Escalation) GetAdditionalRecipients() []EmailRecipient`

GetAdditionalRecipients returns the AdditionalRecipients field if non-nil, zero value otherwise.

### GetAdditionalRecipientsOk

`func (o *Escalation) GetAdditionalRecipientsOk() (*[]EmailRecipient, bool)`

GetAdditionalRecipientsOk returns a tuple with the AdditionalRecipients field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetAdditionalRecipients

`func (o *Escalation) SetAdditionalRecipients(v []EmailRecipient)`

SetAdditionalRecipients sets AdditionalRecipients field to given value.

### HasAdditionalRecipients

`func (o *Escalation) HasAdditionalRecipients() bool`

HasAdditionalRecipients returns a boolean if a field has been set.

### SetAdditionalRecipientsNil

`func (o *Escalation) SetAdditionalRecipientsNil(b bool)`

 SetAdditionalRecipientsNil sets the value for AdditionalRecipients to be an explicit nil

### UnsetAdditionalRecipients
`func (o *Escalation) UnsetAdditionalRecipients()`

UnsetAdditionalRecipients ensures that no value is present for AdditionalRecipients, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


