package beanq

func (c *Client) Migrate() error {

	jsonStr := c.config.ToJson()
	// hide the ‘config’ parameter and prohibit manually passing in this parameter
	migrationCmd.Flags().String(cmdConfigKeyName, jsonStr, "")
	_ = migrationCmd.Flags().MarkHidden(cmdConfigKeyName)
	// migration type: up | down
	migrationCmd.Flags().String("action", "up", "performed action")
	migrationCmd.Flags().String("file", "", "migration files")
	// If you want to perform a database migration, use the command [run migration --action=down].
	//The default action is up.
	if err := Execute(); err != nil {
		return err
	}
	return nil
}
