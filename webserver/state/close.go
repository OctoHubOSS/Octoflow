//  Copyright (C) 2026 NodeByte LTD

package state

func Close() {
	Logger.Info("Closing service [postgres]")
	Pool.Close()

	Logger.Info("Closing service [discord]")
	Discord.Close()
}
