package main

import (
	"time"

	"github.com/landoop/lenses-go"

	"github.com/landoop/bite"
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
				return err
			}

			// return printJSON(cmd, alerts)
			return bite.PrintObject(cmd, alerts)
		},
	}

	cmd.Flags().BoolVar(&sse, "live", false, "--live Enables real-time push alert notifications")

	bite.CanPrintJSON(cmd)

	return cmd
}

func newAlertGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "alert",
		Short:            "Work with alerts",
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
		Example:          `alert register ./alert.yml or alert register --alert=1000 --startsAt="2018-03-27T21:23:23.634+02:00" --endsAt=... --source="" --summary="Broker on 1 is down" --docs="" --category="Infrastructure" --severity="HIGH" --instance="instance101" --generator="http://lenses"`,
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
				return err
			}

			return bite.PrintInfo(cmd, "Alert with ID %d registered", alert.AlertID)
		},
	}

	cmd.Flags().IntVar(&alert.AlertID, "alert", 0, "--alert=1000")
	cmd.Flags().StringVar(&alert.StartsAt, "startsAt", "", `--startsAt="2018-03-27T21:23:23.634+02:00"`)
	cmd.Flags().StringVar(&alert.EndsAt, "endsAt", "", `--endsAt=""`)
	cmd.Flags().StringVar(&alert.Labels.Category, "category", "", `--category="Infrastructure"`)
	cmd.Flags().StringVar(&alert.Labels.Severity, "severity", "", `--severity="HIGH"`)
	cmd.Flags().StringVar(&alert.Labels.Instance, "instance", "", `--instance="instance101"`)
	cmd.Flags().StringVar(&alert.Annotations.Docs, "docs", "", `--docs=""`)
	cmd.Flags().StringVar(&alert.Annotations.Source, "source", "", `--source=""`)
	cmd.Flags().StringVar(&alert.Annotations.Summary, "summary", "", `--summary="Broken on 1 is down"`)
	cmd.Flags().StringVar(&alert.GeneratorURL, "generator", "", `--generator="http://lenses" is a unique URL identifying the creator of this alert. It matches AlertManager requirements for providing this field`)

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
			bite.FriendlyError(cmd, errResourceNotFoundMessage, "alert setting with id %d does not exist", id)

			if mustEnable {
				if err := client.EnableAlertSetting(id); err != nil {
					return err
				}

				return bite.PrintInfo(cmd, "Alert setting %d enabled", id)
			}

			settings, err := client.GetAlertSetting(id)
			if err != nil {
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
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to retrieve conditions, alert setting with id %d does not exist", alertID)
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
		Short:            "Work with an alert setting's condition",
		Example:          `alert setting condition delete --alert=1001 --condition="28bbad2b-69bb-4c01-8e37-28e2e7083aa9"`,
		TraverseChildren: true,
		SilenceErrors:    true,
	}

	rootSub.AddCommand(newCreateOrUpdateAlertSettingConditionCommand())
	rootSub.AddCommand(newDeleteAlertSettingConditionCommand())

	return rootSub
}

type alertSettingConditionPayload struct {
	AlertID   int    `json:"alert" yaml:"Alert"`
	Condition string `json:"condition" yaml:"Condition"`
}

func newCreateOrUpdateAlertSettingConditionCommand() *cobra.Command {
	var cond alertSettingConditionPayload

	cmd := &cobra.Command{
		Use:              "set",
		Aliases:          []string{"create", "update"},
		Short:            "Create or Update an alert setting's condition",
		Example:          `alert setting condition set --alert=1001 --condition="lag >= 100000 on group group and topic topicA" or alert setting condition set ./alert_cond.yml`,
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := bite.CheckRequiredFlags(cmd, bite.FlagPair{"alert": cond.AlertID, "condition": cond.Condition}); err != nil {
				return err
			}

			err := client.CreateOrUpdateAlertSettingCondition(cond.AlertID, cond.Condition)
			if err != nil {
				return err
			}

			return bite.PrintInfo(cmd, "Condition for alert setting %d updated", cond.AlertID)
		},
	}

	cmd.Flags().IntVar(&cond.AlertID, "alert", 0, "--alert=1001")
	cmd.Flags().StringVar(&cond.Condition, "condition", "", `--condition="lag >= 100000 on group group and topic topicA"`)

	bite.CanBeSilent(cmd)

	bite.Prepend(cmd, bite.FileBind(&cond))

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
				bite.FriendlyError(cmd, errResourceNotFoundMessage, "unable to delete condition, alert setting with id %d or condition with UUID '%s' does not exist", alertID, conditionUUID)
				return err
			}

			return bite.PrintInfo(cmd, "Condition '%s' for alert setting %d deleted", conditionUUID, alertID)
		},
	}

	cmd.Flags().IntVar(&alertID, "alert", 0, "--alert=1001")
	cmd.MarkFlagRequired("alert")
	cmd.Flags().StringVar(&conditionUUID, "condition", "", `--condition="28bbad2b-69bb-4c01-8e37-28e2e7083aa9"`)
	cmd.MarkFlagRequired("condition")
	bite.CanBeSilent(cmd)

	return cmd
}
