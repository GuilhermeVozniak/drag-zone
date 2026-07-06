// Package builtin implements DragZone's built-in actions.
package builtin

import "dragzone/internal/actions"

// RegisterAll registers every built-in action with the registry.
func RegisterAll(reg *actions.Registry) {
	reg.Register(Folder{})
	reg.Register(OpenApplication{})
	reg.Register(AirDrop{})
	reg.Register(CopyToClipboard{})
	reg.Register(ZipFiles{})
	reg.Register(Trash{})
	reg.Register(InstallApp{})
	reg.Register(SaveText{})
	reg.Register(PrintFiles{})
	reg.Register(ShortenURL{})
	reg.Register(ImgurUpload{})
	reg.Register(FTPUpload{})
	reg.Register(S3Upload{})
	reg.Register(ConvertImages{})
	reg.Register(RemoveImageMetadata{})
	reg.Register(GoogleDriveUpload{})
}
