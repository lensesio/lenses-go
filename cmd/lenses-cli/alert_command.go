package main

import (
	"time"

	"github.com/kataras/golog"
	"github.com/landoop/bite"
	"github.com/landoop/lenses-go"
	"github.com/spf13/cobra"
)

func init() {
	app.AddCommand(newGetAlertsCommand())
	app.AddCommand(newAlertGroupCommand())
}

func newGetAlertsCommand() *cobra.Command {
	var sse bool

	cmd := &cobra.Command{
		Use:              "alerts",
		Short:            "Print the registered alerts",
		Example:          "alerts",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if sse {
				handler := func(alert lenses.Alert) error {
					return bite.PrintObject(cmd, alert) // keep json here?
				}

				return client.GetAlertsLive(handler)
			}

			alerts, err := client.GetAlerts()
			if err != nil {
				golog.Errorf("Failed to retrieve alerts. [%s]", err.Error())
				return err
			}

			//return bite.PrintJSON(cmd, alerts)
			return bite.PrintObject(cmd, alerts)
		},
	}

	cmd.Flags().BoolVar(&sse, "live", false, "Enables real-time push alert notifications")

	bite.CanPrintJSON(cmd)

	return cmd
}

func newAlertGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "alert",
		Short:            "Manage alerts",
		Example:          "alert",
		TraverseChildren: true,
		SilenceErrors:    true,
	}

	root.AddCommand(
		newRegisterAlertCommand(),
		newGetAlertSettingsCommand(),
		newAlertSettingGroupCommand(),
	)

	return root
}

func newRegisterAlertCommand() *cobra.Command {
	var alert lenses.Alert

	cmd := &cobra.Command{
		Use:              "register",
		Short:            "Register an alert",
		Example:          `alert register ./alert.yml or alert register --alert=1000 --startsAt="2018-03-27T21:23:23.634+02:00" --endsAt=... --source="" --summary="Broker on 1 is down" --docs="" --category="Infrastructure" --severity="HIGH" --instance="instance101" --generator="https://lenses"`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if alert.StartsAt == "" {
				alert.StartsAt = time.Now().Format(time.RFC3339)
			}

			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"alert": alert.AlertID, "severity": alert.Labels.Severity, "summary": alert.Annotations.Summary, "generatorURL": alert.GeneratorURL}); err != nil {
				return err
			}

			err := client.RegisterAlert(alert)
			if err != nil {
				golog.Errorf("Failed to register alert. [%s]", err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Alert with ID [%d] registered", alert.AlertID)
		},
	}

	cmd.Flags().IntVar(&alert.AlertID, "alert", 0, "Alert ID to register")
	cmd.Flags().StringVar(&alert.StartsAt, "startsAt", "", `Alert start time .e.g. "2018-03-27T21:23:23.634+02:00"`)
	cmd.Flags().StringVar(&alert.EndsAt, "endsAt", "", `Alert end time"`)
	cmd.Flags().StringVar(&alert.Labels.Category, "category", "", `Category .e.g. "Infrastructure"`)
	cmd.Flags().StringVar(&alert.Labels.Severity, "severity", "", `Severity .e.g. "HIGH"`)
	cmd.Flags().StringVar(&alert.Labels.Instance, "instance", "", `Instance .e.g. "instance101"`)
	cmd.Flags().StringVar(&alert.Annotations.Docs, "docs", "", `Documentation text`)
	cmd.Flags().StringVar(&alert.Annotations.Source, "source", "", `Source of the alert`)
	cmd.Flags().StringVar(&alert.Annotations.Summary, "summary", "", `Alert summary`)
	cmd.Flags().StringVar(&alert.GeneratorURL, "generator", "", `Unique URL identifying the creator of this alert. It matches AlertManager requirements for providing this field`)

	bite.Prepend(cmd, bite.FileBind(&alert))

	bite.CanBeSilent(cmd)

	return cmd
}

func newGetAlertSettingsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "settings",
		Short:            "Print all alert settings",
		Example:          "alert settings",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := client.GetAlertSettings()
			if err != nil {
				return err
			}

			// force json, may contains conditions that are easier to be seen in json format.
			return bite.PrintJSON(cmd, settings)
		},
	}

	bite.CanPrintJSON(cmd)

	return cmd
}

func newAlertSettingGroupCommand() *cobra.Command {
	var (
		id         int
		mustEnable bool
	)

	root := &cobra.Command{
		Use:              "setting",
		Short:            "Print or enable a specific alert setting based on ID",
		Example:          "alert setting --id=1001 [--enable]",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if mustEnable {
				if err := client.EnableAlertSetting(id); err != nil {
					return err
				}

				return bite.PrintInfo(cmd, "Alert setting [%d] enabled", id)
			}

			settings, err := client.GetAlertSetting(id)
			if err != nil {
				golog.Errorf("Failed to retrieve alert [%d]. [%s]", id, err.Error())
				return err
			}

			// force json, may contains conditions that are easier to be seen in json format.
			return bite.PrintJSON(cmd, settings)
		},
	}

	root.Flags().IntVar(&id, "id", 0, "--id=1001")
	root.MarkFlagRequired("id")

	root.Flags().BoolVar(&mustEnable, "enable", false, "--enable")

	bite.CanPrintJSON(root)
	bite.CanBeSilent(root)

	root.AddCommand(newGetAlertSettingConditionsCommand())
	root.AddCommand(newAlertSettingConditionGroupCommand())

	return root
}

func newGetAlertSettingConditionsCommand() *cobra.Command {
	var alertID int

	cmd := &cobra.Command{
		Use:              "conditions",
		Short:            "Print alert setting's conditions",
		Example:          "alert setting conditions --alert=1001",
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			conds, err := client.GetAlertSettingConditions(alertID)
			if err != nil {
				golog.Errorf("Failed to retrieve alert setting conditions for [%d]. [%s]", alertID, err.Error())
				return err
			}

			// force-json
			return bite.PrintJSON(cmd, conds)
		},
	}

	cmd.Flags().IntVar(&alertID, "alert", 0, "--alert=1001")
	cmd.MarkFlagRequired("alert")

	bite.CanBeSilent(cmd)
	bite.CanPrintJSON(cmd)

	return cmd
}

func newAlertSettingConditionGroupCommand() *cobra.Command {
	rootSub := &cobra.Command{
		Use:              "condition",
		Short:            "Manage alert setting's condition",
		Example:          `alert setting condition delete --alert=1001 --condition="28bbad2b-69bb-4c01-8e37-28e2e7083aa9"`,
		TraverseChildren: true,
		SilenceErrors:    true,
	}

	rootSub.AddCommand(newCreateOrUpdateAlertSettingConditionCommand())
	rootSub.AddCommand(newDeleteAlertSettingConditionCommand())

	return rootSub
}

// AlertSettingConditionPayloads is the payload for creating alert setttings
type AlertSettingConditionPayloads struct {
	AlertID    int      `json:"alert" yaml:"alert"`
	Conditions []string `json:"conditions" yaml:"conditions"`
}

// AlertSettingConditionPayload is the payload for creating alert setttings
type AlertSettingConditionPayload struct {
	AlertID   int    `json:"alert" yaml:"alert"`
	Condition string `json:"condition" yaml:"condition"`
}

func newCreateOrUpdateAlertSettingConditionCommand() *cobra.Command {
	var conds AlertSettingConditionPayloads
	var cond AlertSettingConditionPayload

	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"},
		Short:            "Create or Update an alert setting's condition or load from file",
		Example:          `alert setting condition set --alert=1001 --condition="lag >= 100000or alert setting condition set ./alert_cond.yml`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {

			if len(conds.Conditions) > 0 {
				alertID := conds.AlertID
				for _, condition := range conds.Conditions {
					err := client.CreateOrUpdateAlertSettingCondition(alertID, condition)
					if err != nil {
						golog.Errorf("Failed to creating/updating alert setting condition [%s]. [%s]", condition, err.Error())
						return err
					}
					bite.PrintInfo(cmd, "Condition [%s] added", condition)
				}
				return nil
			}

			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"alert": cond.AlertID, "condition": cond.Condition}); err != nil {
				return err
			}

			err := client.CreateOrUpdateAlertSettingCondition(cond.AlertID, cond.Condition)
			if err != nil {
				golog.Errorf("Failed to creating/updating alert setting condition [%s]. [%s]", cond.Condition, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Condition [%s] added", cond.Condition)
		},
	}

	cmd.Flags().IntVar(&cond.AlertID, "alert", 0, "Alert ID")
	cmd.Flags().StringVar(&cond.Condition, "condition", "", `Alert condition .e.g. "lag >= 100000 on group group and topic topicA"`)

	bite.CanBeSilent(cmd)

	bite.Prepend(cmd, bite.FileBind(&conds))

	return cmd
}

func newDeleteAlertSettingConditionCommand() *cobra.Command {
	var (
		alertID       int
		conditionUUID string
	)

	cmd := &cobra.Command{
		Use:              "delete",
		Short:            "Delete an alert setting's condition",
		Example:          `alert setting condition delete --alert=1001 --condition="28bbad2b-69bb-4c01-8e37-28e2e7083aa9"`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := client.DeleteAlertSettingCondition(alertID, conditionUUID)
			if err != nil {
				golog.Errorf("Failed to deleting alert setting condition [%s]. [%s]", conditionUUID, err.Error())
				return err
			}

			return bite.PrintInfo(cmd, "Condition [%s] for alert setting [%d] deleted", conditionUUID, alertID)
		},
	}

	cmd.Flags().IntVar(&alertID, "alert", 0, "Alert ID")
	cmd.MarkFlagRequired("alert")
	cmd.Flags().StringVar(&conditionUUID, "condition", "", `Alert condition uuid .e.g. "28bbad2b-69bb-4c01-8e37-28e2e7083aa9"`)
	cmd.MarkFlagRequired("condition")
	bite.CanBeSilent(cmd)

	return cmd
}
