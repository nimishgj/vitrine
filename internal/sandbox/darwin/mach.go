// Package darwin implements the native macOS backend: a separate OS user
// running under a Seatbelt (sandbox-exec) profile.
package darwin

// MachServices are the launchd services a sandboxed process needs to
// function at all (DNS, locale, trust evaluation, notifications). The list
// is pruned by the probe: remove an entry, run the integration test, keep it
// out if everything still passes.
func MachServices() []string {
	return []string{
		"com.apple.system.logger",
		"com.apple.logd",
		"com.apple.system.notification_center",
		"com.apple.system.opendirectoryd.libinfo",
		"com.apple.system.opendirectoryd.membership",
		"com.apple.SecurityServer",
		"com.apple.trustd",
		"com.apple.trustd.agent",
		"com.apple.networkd",
		"com.apple.nehelper",
		"com.apple.nesessionmanager.content-filter",
		"com.apple.dnssd.service",
		"com.apple.coreservices.launchservicesd",
		"com.apple.distributed_notifications@Uv3",
		"com.apple.cfprefsd.daemon",
		"com.apple.cfprefsd.agent",
		"com.apple.FSEvents",
		"com.apple.fonts",
		"com.apple.lsd.mapdb",
		"com.apple.system.DirectoryService.libinfo_v1",
	}
}
