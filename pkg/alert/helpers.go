package alert

import (
	"encoding/json"
	"fmt"

	"github.com/lensesio/lenses-go/v5/pkg/api"
)

type Client interface {
	HasAlertSettingsV2Endpoints() (bool, error)
	GetAlertSettingV1(id int) (api.AlertSettingV1, error)
	GetAlertSettingV2(id int) (api.AlertRuleV2, error)
}

var _ Client = &api.Client{}

// GetConsumerAlertSettings returns ConsumerAlertSettings. It uses the v1 or v2
// endpoint depending on its availability.
func GetConsumerAlertSettings(client Client) (api.ConsumerAlertSettings, error) {
	useV2, err := client.HasAlertSettingsV2Endpoints()
	if err != nil {
		return api.ConsumerAlertSettings{}, fmt.Errorf("check v2 endpoint availability: %w", err)
	}
	if useV2 {
		return getConsumerAlertSettingsV1FromV2(client)
	}
	return getConsumerAlertSettingsV1(client)
}

// GetProducerAlertSettings returns ConsumerAlertSettings. It uses the v1 or v2
// endpoint depending on its availability.
func GetProducerAlertSettings(client Client) (api.ProducerAlertSettings, error) {
	useV2, err := client.HasAlertSettingsV2Endpoints()
	if err != nil {
		return api.ProducerAlertSettings{}, fmt.Errorf("check v2 endpoint availability: %w", err)
	}
	if useV2 {
		return getProducerAlertSettingsV1FromV2(client)
	}
	return getProducerAlertSettingsV1(client)
}

func getConsumerAlertSettingsV1(client Client) (api.ConsumerAlertSettings, error) {
	settings, err := client.GetAlertSettingV1(2000)
	if err != nil {
		return api.ConsumerAlertSettings{}, err
	}

	consumerAlertSettings := api.ConsumerAlertSettings{
		ID:          settings.ID,
		Description: settings.Description,
	}

	// iterate over the consumer condition details
	for _, condDetail := range settings.ConditionDetails {
		jsonStringCondition, _ := json.Marshal(condDetail.ConditionDsl)

		consumerAlertConditionDetail := api.ConsumerAlertConditionRequestv1{}
		if err := json.Unmarshal(jsonStringCondition, &consumerAlertConditionDetail.Condition); err != nil {
			return api.ConsumerAlertSettings{}, fmt.Errorf("unmarshal condition: %w", err)
		}

		// iterate channels of a condition detail
		for _, chann := range condDetail.Channels {
			consumerAlertConditionDetail.Channels = append(consumerAlertConditionDetail.Channels, chann.Name)
		}

		consumerAlertSettings.ConditionDetails = append(consumerAlertSettings.ConditionDetails, consumerAlertConditionDetail)
	}

	return consumerAlertSettings, nil
}

func getProducerAlertSettingsV1(client Client) (api.ProducerAlertSettings, error) {
	settings, err := client.GetAlertSettingV1(5000)
	if err != nil {
		return api.ProducerAlertSettings{}, err
	}

	producerAlertSettings := api.ProducerAlertSettings{
		ID:          settings.ID,
		Description: settings.Description,
	}

	// iterate over the data produced condition details
	for _, condDetail := range settings.ConditionDetails {
		jsonStringCondition, _ := json.Marshal(condDetail.ConditionDsl)

		producerAlertConditionDetail := api.AlertConditionRequestv1{}
		if err := json.Unmarshal(jsonStringCondition, &producerAlertConditionDetail.Condition); err != nil {
			return api.ProducerAlertSettings{}, fmt.Errorf("unmarshal condition: %w", err)
		}

		// iterate channels of a condition detail
		for _, chann := range condDetail.Channels {
			producerAlertConditionDetail.Channels = append(producerAlertConditionDetail.Channels, chann.Name)
		}

		producerAlertSettings.ConditionDetails = append(producerAlertSettings.ConditionDetails, producerAlertConditionDetail)
	}

	return producerAlertSettings, nil
}

func getConsumerAlertSettingsV1FromV2(client Client) (api.ConsumerAlertSettings, error) {
	settings, err := client.GetAlertSettingV2(2000)
	if err != nil {
		return api.ConsumerAlertSettings{}, err
	}

	if settings.Details.AlertType != api.AlertTypeConditional {
		return api.ConsumerAlertSettings{}, fmt.Errorf("expected a conditional alert type; got: %q", settings.Details.AlertType)
	}

	consumerAlertSettings := api.ConsumerAlertSettings{
		ID:          settings.ID,
		Description: settings.Description,
	}

	for id, con := range settings.Details.Conditional.Conditions {
		detail := api.ConsumerAlertConditionRequestv1{}
		if err := json.Unmarshal(con.ConditionDsl, &detail.Condition); err != nil {
			return api.ConsumerAlertSettings{}, fmt.Errorf("unmarshal condition dsl %q: %w", id, err)
		}

		for _, chann := range con.Channels {
			detail.Channels = append(detail.Channels, chann.Name)
		}

		consumerAlertSettings.ConditionDetails = append(consumerAlertSettings.ConditionDetails, detail)
	}

	return consumerAlertSettings, nil
}

func getProducerAlertSettingsV1FromV2(client Client) (api.ProducerAlertSettings, error) {
	settings, err := client.GetAlertSettingV2(5000)
	if err != nil {
		return api.ProducerAlertSettings{}, fmt.Errorf("get alert setting v2: %w", err)
	}

	if settings.Details.AlertType != api.AlertTypeConditional {
		return api.ProducerAlertSettings{}, fmt.Errorf("expected a conditional alert type; got: %q", settings.Details.AlertType)
	}

	prodAlert := api.ProducerAlertSettings{
		ID:          settings.ID,
		Description: settings.Description,
	}

	for id, con := range settings.Details.Conditional.Conditions {
		detail := api.AlertConditionRequestv1{}
		if err := json.Unmarshal(con.ConditionDsl, &detail.Condition); err != nil {
			return api.ProducerAlertSettings{}, fmt.Errorf("unmarshal condition dsl %q: %w", id, err)
		}

		for _, chann := range con.Channels {
			detail.Channels = append(detail.Channels, chann.Name)
		}

		prodAlert.ConditionDetails = append(prodAlert.ConditionDetails, detail)
	}

	return prodAlert, nil
}
