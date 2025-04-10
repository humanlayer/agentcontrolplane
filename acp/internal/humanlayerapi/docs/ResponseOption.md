# ResponseOption

## Properties

Name | Type | Description | Notes
------------ | ------------- | ------------- | -------------
**Name** | **string** |  | 
**Title** | Pointer to **NullableString** |  | [optional] 
**Description** | Pointer to **NullableString** |  | [optional] 
**PromptFill** | Pointer to **NullableString** |  | [optional] 
**Interactive** | Pointer to **bool** |  | [optional] [default to false]

## Methods

### NewResponseOption

`func NewResponseOption(name string, ) *ResponseOption`

NewResponseOption instantiates a new ResponseOption object
This constructor will assign default values to properties that have it defined,
and makes sure properties required by API are set, but the set of arguments
will change when the set of required properties is changed

### NewResponseOptionWithDefaults

`func NewResponseOptionWithDefaults() *ResponseOption`

NewResponseOptionWithDefaults instantiates a new ResponseOption object
This constructor will only assign default values to properties that have it defined,
but it doesn't guarantee that properties required by API are set

### GetName

`func (o *ResponseOption) GetName() string`

GetName returns the Name field if non-nil, zero value otherwise.

### GetNameOk

`func (o *ResponseOption) GetNameOk() (*string, bool)`

GetNameOk returns a tuple with the Name field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetName

`func (o *ResponseOption) SetName(v string)`

SetName sets Name field to given value.


### GetTitle

`func (o *ResponseOption) GetTitle() string`

GetTitle returns the Title field if non-nil, zero value otherwise.

### GetTitleOk

`func (o *ResponseOption) GetTitleOk() (*string, bool)`

GetTitleOk returns a tuple with the Title field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetTitle

`func (o *ResponseOption) SetTitle(v string)`

SetTitle sets Title field to given value.

### HasTitle

`func (o *ResponseOption) HasTitle() bool`

HasTitle returns a boolean if a field has been set.

### SetTitleNil

`func (o *ResponseOption) SetTitleNil(b bool)`

 SetTitleNil sets the value for Title to be an explicit nil

### UnsetTitle
`func (o *ResponseOption) UnsetTitle()`

UnsetTitle ensures that no value is present for Title, not even an explicit nil
### GetDescription

`func (o *ResponseOption) GetDescription() string`

GetDescription returns the Description field if non-nil, zero value otherwise.

### GetDescriptionOk

`func (o *ResponseOption) GetDescriptionOk() (*string, bool)`

GetDescriptionOk returns a tuple with the Description field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetDescription

`func (o *ResponseOption) SetDescription(v string)`

SetDescription sets Description field to given value.

### HasDescription

`func (o *ResponseOption) HasDescription() bool`

HasDescription returns a boolean if a field has been set.

### SetDescriptionNil

`func (o *ResponseOption) SetDescriptionNil(b bool)`

 SetDescriptionNil sets the value for Description to be an explicit nil

### UnsetDescription
`func (o *ResponseOption) UnsetDescription()`

UnsetDescription ensures that no value is present for Description, not even an explicit nil
### GetPromptFill

`func (o *ResponseOption) GetPromptFill() string`

GetPromptFill returns the PromptFill field if non-nil, zero value otherwise.

### GetPromptFillOk

`func (o *ResponseOption) GetPromptFillOk() (*string, bool)`

GetPromptFillOk returns a tuple with the PromptFill field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetPromptFill

`func (o *ResponseOption) SetPromptFill(v string)`

SetPromptFill sets PromptFill field to given value.

### HasPromptFill

`func (o *ResponseOption) HasPromptFill() bool`

HasPromptFill returns a boolean if a field has been set.

### SetPromptFillNil

`func (o *ResponseOption) SetPromptFillNil(b bool)`

 SetPromptFillNil sets the value for PromptFill to be an explicit nil

### UnsetPromptFill
`func (o *ResponseOption) UnsetPromptFill()`

UnsetPromptFill ensures that no value is present for PromptFill, not even an explicit nil
### GetInteractive

`func (o *ResponseOption) GetInteractive() bool`

GetInteractive returns the Interactive field if non-nil, zero value otherwise.

### GetInteractiveOk

`func (o *ResponseOption) GetInteractiveOk() (*bool, bool)`

GetInteractiveOk returns a tuple with the Interactive field if it's non-nil, zero value otherwise
and a boolean to check if the value has been set.

### SetInteractive

`func (o *ResponseOption) SetInteractive(v bool)`

SetInteractive sets Interactive field to given value.

### HasInteractive

`func (o *ResponseOption) HasInteractive() bool`

HasInteractive returns a boolean if a field has been set.


[[Back to Model list]](../README.md#documentation-for-models) [[Back to API list]](../README.md#documentation-for-api-endpoints) [[Back to README]](../README.md)


