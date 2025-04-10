# ContactChannelOutput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Slack** | Pointer to [**NullableSlackContactChannelOutput**](SlackContactChannelOutput.md) |  | [optional] 
**Sms** | Pointer to [**NullableSMSContactChannel**](SMSContactChannel.md) |  | [optional] 
**Whatsapp** | Pointer to [**NullableWhatsAppContactChannel**](WhatsAppContactChannel.md) |  | [optional] 
**Email** | Pointer to [**NullableEmailContactChannel**](EmailContactChannel.md) |  | [optional] 

## Methods

### NewContactChannelOutput

`func NewContactChannelOutput() *ContactChannelOutput`

NewContactChannelOutput instantiates a new ContactChannelOutput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewContactChannelOutputWithDefaults

`func NewContactChannelOutputWithDefaults() *ContactChannelOutput`

NewContactChannelOutputWithDefaults instantiates a new ContactChannelOutput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSlack

`func (o *ContactChannelOutput) GetSlack() SlackContactChannelOutput`

GetSlack returns the Slack field if non-nil, zero value otherwise.

### GetSlackOk

`func (o *ContactChannelOutput) GetSlackOk() (*SlackContactChannelOutput, bool)`

GetSlackOk returns a tuple with the Slack field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSlack

`func (o *ContactChannelOutput) SetSlack(v SlackContactChannelOutput)`

SetSlack sets Slack field to given value.

### HasSlack

`func (o *ContactChannelOutput) HasSlack() bool`

HasSlack returns a boolean if a field has been set.

### SetSlackNil

`func (o *ContactChannelOutput) SetSlackNil(b bool)`

 SetSlackNil sets the value for Slack to be an explicit nil

### UnsetSlack
`func (o *ContactChannelOutput) UnsetSlack()`

UnsetSlack ensures that no value is present for Slack, not even an explicit nil
### GetSms

`func (o *ContactChannelOutput) GetSms() SMSContactChannel`

GetSms returns the Sms field if non-nil, zero value otherwise.

### GetSmsOk

`func (o *ContactChannelOutput) GetSmsOk() (*SMSContactChannel, bool)`

GetSmsOk returns a tuple with the Sms field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSms

`func (o *ContactChannelOutput) SetSms(v SMSContactChannel)`

SetSms sets Sms field to given value.

### HasSms

`func (o *ContactChannelOutput) HasSms() bool`

HasSms returns a boolean if a field has been set.

### SetSmsNil

`func (o *ContactChannelOutput) SetSmsNil(b bool)`

 SetSmsNil sets the value for Sms to be an explicit nil

### UnsetSms
`func (o *ContactChannelOutput) UnsetSms()`

UnsetSms ensures that no value is present for Sms, not even an explicit nil
### GetWhatsapp

`func (o *ContactChannelOutput) GetWhatsapp() WhatsAppContactChannel`

GetWhatsapp returns the Whatsapp field if non-nil, zero value otherwise.

### GetWhatsappOk

`func (o *ContactChannelOutput) GetWhatsappOk() (*WhatsAppContactChannel, bool)`

GetWhatsappOk returns a tuple with the Whatsapp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWhatsapp

`func (o *ContactChannelOutput) SetWhatsapp(v WhatsAppContactChannel)`

SetWhatsapp sets Whatsapp field to given value.

### HasWhatsapp

`func (o *ContactChannelOutput) HasWhatsapp() bool`

HasWhatsapp returns a boolean if a field has been set.

### SetWhatsappNil

`func (o *ContactChannelOutput) SetWhatsappNil(b bool)`

 SetWhatsappNil sets the value for Whatsapp to be an explicit nil

### UnsetWhatsapp
`func (o *ContactChannelOutput) UnsetWhatsapp()`

UnsetWhatsapp ensures that no value is present for Whatsapp, not even an explicit nil
### GetEmail

`func (o *ContactChannelOutput) GetEmail() EmailContactChannel`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *ContactChannelOutput) GetEmailOk() (*EmailContactChannel, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *ContactChannelOutput) SetEmail(v EmailContactChannel)`

SetEmail sets Email field to given value.

### HasEmail

`func (o *ContactChannelOutput) HasEmail() bool`

HasEmail returns a boolean if a field has been set.

### SetEmailNil

`func (o *ContactChannelOutput) SetEmailNil(b bool)`

 SetEmailNil sets the value for Email to be an explicit nil

### UnsetEmail
`func (o *ContactChannelOutput) UnsetEmail()`

UnsetEmail ensures that no value is present for Email, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


