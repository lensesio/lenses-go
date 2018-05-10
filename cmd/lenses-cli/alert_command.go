package main

import (
	"fmt"
	"time"

	"github.com/landoop/lenses-go"

	"github.com/spf13/cobra"
)

func init() {
	rootCmd.AddCommand(newGetAlertsCommand())
	rootCmd.AddCommand(newAlertGroupCommand())
}

func newGetAlertsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "alerts",
		Short:            "Print the registered alerts",
		Example:          exampleString("alerts"),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			alerts, err := client.GetAlerts()
			if err != nil {
				return err
			}

			return printJSON(cmd, alerts)
		},
	}

	canPrintJSON(cmd)

	return cmd
}

func newAlertGroupCommand() *cobra.Command {
	root := &cobra.Command{
		Use:              "alert",
		Short:            "Work with alerts",
		Example:          exampleString("alert"),
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
		Example:          exampleString(`alert register ./alert.yml or alert register --alert=1000 --startsAt="2018-03-27T21:23:23.634+02:00" --endsAt=... --source="" --summary="Broker on 1 is down" --docs="" --category="Infrastructure" --severity="HIGH" --instance="instance101" --generator="http://lenses"`),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if alert.StartsAt == "" {
				alert.StartsAt = time.Now().Format(time.RFC3339)
			}

			if err := checkRequiredFlags(cmd, flags{"alert": alert.AlertID, "severity": alert.Labels.Severity, "summary": alert.Annotations.Summary, "generatorURL": alert.GeneratorURL}); err != nil {
				return err
			}

			err := client.RegisterAlert(alert)
			if err != nil {
				return err
			}

			return echo(cmd, "Alert with ID %d registered", alert.AlertID)
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

	shouldTryLoadFile(cmd, &alert)
	canBeSilent(cmd)

	return cmd
}

func newGetAlertSettingsCommand() *cobra.Command {
	cmd := &cobra.Command{
		Use:              "settings",
		Short:            "Print all alert settings",
		Example:          exampleString("alert settings"),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			settings, err := client.GetAlertSettings()
			if err != nil {
				return err
			}

			return printJSON(cmd, settings)
		},
	}

	canPrintJSON(cmd)

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
		Example:          exampleString("alert setting --id=1001 [--enable]"),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			errResourceNotFoundMessage = fmt.Sprintf("alert setting with id %d does not exist", id)

			if mustEnable {
				if err := client.EnableAlertSetting(id); err != nil {
					return err
				}

				return echo(cmd, "Alert setting %d enabled", id)
			}

			settings, err := client.GetAlertSetting(id)
			if err != nil {
				return err
			}

			return printJSON(cmd, settings)
		},
	}

	root.Flags().IntVar(&id, "id", 0, "--id=1001")
	root.MarkFlagRequired("id")

	root.Flags().BoolVar(&mustEnable, "enable", false, "--enable")

	canBeSilent(root)
	canPrintJSON(root)

	root.AddCommand(newGetAlertSettingConditionsCommand())
	root.AddCommand(newAlertSettingConditionGroupCommand())

	return root
}

func newGetAlertSettingConditionsCommand() *cobra.Command {
	var alertID int

	cmd := &cobra.Command{
		Use:              "conditions",
		Short:            "Print alert setting's conditions",
		Example:          exampleString("alert setting conditions --id=1001"),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			conds, err := client.GetAlertSettingConditions(alertID)
			if err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to retrieve conditions, alert setting with id %d does not exist", alertID)
				return err
			}

			return printJSON(cmd, conds)
		},
	}

	cmd.Flags().IntVar(&alertID, "alert", 0, "--alert=1001")
	cmd.MarkFlagRequired("alert")

	canBeSilent(cmd)
	canPrintJSON(cmd)

	return cmd
}

func newAlertSettingConditionGroupCommand() *cobra.Command {
	rootSub := &cobra.Command{
		Use:              "condition",
		Short:            "Work with an alert setting's condition",
		Example:          exampleString(`alert setting condition delete --alert=1001 --condition="28bbad2b-69bb-4c01-8e37-28e2e7083aa9"`),
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
		Example:          exampleString(`alert setting condition set --alert=1001 --condition="lag >= 100000 on group group and topic topicA" or alert setting condition set ./alert_cond.yml`),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			if err := checkRequiredFlags(cmd, flags{"alert": cond.AlertID, "condition": cond.Condition}); err != nil {
				return err
			}

			err := client.CreateOrUpdateAlertSettingCondition(cond.AlertID, cond.Condition)
			if err != nil {
				return err
			}

			return echo(cmd, "Condition for alert setting %d updated", cond.AlertID)
		},
	}

	cmd.Flags().IntVar(&cond.AlertID, "alert", 0, "--alert=1001")
	cmd.Flags().StringVar(&cond.Condition, "condition", "", `--condition="lag >= 100000 on group group and topic topicA"`)

	canBeSilent(cmd)
	shouldTryLoadFile(cmd, &cond)

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
		Example:          exampleString(`alert setting condition delete --alert=1001 --condition="28bbad2b-69bb-4c01-8e37-28e2e7083aa9"`),
		TraverseChildren: true,
		SilenceErrors:    true,
		RunE: func(cmd *cobra.Command, args []string) error {
			err := client.DeleteAlertSettingCondition(alertID, conditionUUID)
			if err != nil {
				errResourceNotFoundMessage = fmt.Sprintf("unable to delete condition, alert setting with id %d or condition with UUID '%s' does not exist", alertID, conditionUUID)
				return err
			}

			return echo(cmd, "Condition '%s' for alert setting %d deleted", conditionUUID, alertID)
		},
	}

	cmd.Flags().IntVar(&alertID, "alert", 0, "--alert=1001")
	cmd.MarkFlagRequired("alert")
	cmd.Flags().StringVar(&conditionUUID, "condition", "", `--condition="28bbad2b-69bb-4c01-8e37-28e2e7083aa9"`)
	cmd.MarkFlagRequired("condition")

	canBeSilent(cmd)

	return cmd
}
