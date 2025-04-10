# ContactChannelInput

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Slack** | Pointer to [**NullableSlackContactChannelInput**](SlackContactChannelInput.md) |  | [optional] 
**Sms** | Pointer to [**NullableSMSContactChannel**](SMSContactChannel.md) |  | [optional] 
**Whatsapp** | Pointer to [**NullableWhatsAppContactChannel**](WhatsAppContactChannel.md) |  | [optional] 
**Email** | Pointer to [**NullableEmailContactChannel**](EmailContactChannel.md) |  | [optional] 

## Methods

### NewContactChannelInput

`func NewContactChannelInput() *ContactChannelInput`

NewContactChannelInput instantiates a new ContactChannelInput object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewContactChannelInputWithDefaults

`func NewContactChannelInputWithDefaults() *ContactChannelInput`

NewContactChannelInputWithDefaults instantiates a new ContactChannelInput object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetSlack

`func (o *ContactChannelInput) GetSlack() SlackContactChannelInput`

GetSlack returns the Slack field if non-nil, zero value otherwise.

### GetSlackOk

`func (o *ContactChannelInput) GetSlackOk() (*SlackContactChannelInput, bool)`

GetSlackOk returns a tuple with the Slack field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSlack

`func (o *ContactChannelInput) SetSlack(v SlackContactChannelInput)`

SetSlack sets Slack field to given value.

### HasSlack

`func (o *ContactChannelInput) HasSlack() bool`

HasSlack returns a boolean if a field has been set.

### SetSlackNil

`func (o *ContactChannelInput) SetSlackNil(b bool)`

 SetSlackNil sets the value for Slack to be an explicit nil

### UnsetSlack
`func (o *ContactChannelInput) UnsetSlack()`

UnsetSlack ensures that no value is present for Slack, not even an explicit nil
### GetSms

`func (o *ContactChannelInput) GetSms() SMSContactChannel`

GetSms returns the Sms field if non-nil, zero value otherwise.

### GetSmsOk

`func (o *ContactChannelInput) GetSmsOk() (*SMSContactChannel, bool)`

GetSmsOk returns a tuple with the Sms field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetSms

`func (o *ContactChannelInput) SetSms(v SMSContactChannel)`

SetSms sets Sms field to given value.

### HasSms

`func (o *ContactChannelInput) HasSms() bool`

HasSms returns a boolean if a field has been set.

### SetSmsNil

`func (o *ContactChannelInput) SetSmsNil(b bool)`

 SetSmsNil sets the value for Sms to be an explicit nil

### UnsetSms
`func (o *ContactChannelInput) UnsetSms()`

UnsetSms ensures that no value is present for Sms, not even an explicit nil
### GetWhatsapp

`func (o *ContactChannelInput) GetWhatsapp() WhatsAppContactChannel`

GetWhatsapp returns the Whatsapp field if non-nil, zero value otherwise.

### GetWhatsappOk

`func (o *ContactChannelInput) GetWhatsappOk() (*WhatsAppContactChannel, bool)`

GetWhatsappOk returns a tuple with the Whatsapp field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetWhatsapp

`func (o *ContactChannelInput) SetWhatsapp(v WhatsAppContactChannel)`

SetWhatsapp sets Whatsapp field to given value.

### HasWhatsapp

`func (o *ContactChannelInput) HasWhatsapp() bool`

HasWhatsapp returns a boolean if a field has been set.

### SetWhatsappNil

`func (o *ContactChannelInput) SetWhatsappNil(b bool)`

 SetWhatsappNil sets the value for Whatsapp to be an explicit nil

### UnsetWhatsapp
`func (o *ContactChannelInput) UnsetWhatsapp()`

UnsetWhatsapp ensures that no value is present for Whatsapp, not even an explicit nil
### GetEmail

`func (o *ContactChannelInput) GetEmail() EmailContactChannel`

GetEmail returns the Email field if non-nil, zero value otherwise.

### GetEmailOk

`func (o *ContactChannelInput) GetEmailOk() (*EmailContactChannel, bool)`

GetEmailOk returns a tuple with the Email field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetEmail

`func (o *ContactChannelInput) SetEmail(v EmailContactChannel)`

SetEmail sets Email field to given value.

### HasEmail

`func (o *ContactChannelInput) HasEmail() bool`

HasEmail returns a boolean if a field has been set.

### SetEmailNil

`func (o *ContactChannelInput) SetEmailNil(b bool)`

 SetEmailNil sets the value for Email to be an explicit nil

### UnsetEmail
`func (o *ContactChannelInput) UnsetEmail()`

UnsetEmail ensures that no value is present for Email, not even an explicit nil

[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


