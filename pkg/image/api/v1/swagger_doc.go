package v1

// This file contains methods that can be used by the go-restful package to generate Swagger
// documentation for the object types found in 'types.go' This file is automatically generated
// by hack/update-generated-swagger-descriptions.sh and should be run after a full build of OpenShift.
// ==== DO NOT EDIT THIS FILE MANUALLY ====

var map_DockerImageReference = map[string]string{
	"":          "DockerImageReference points to a Docker image.",
	"Registry":  "Registry is the registry that contains the Docker image",
	"Namespace": "Namespace is the namespace that contains the Docker image",
	"Name":      "Name is the name of the Docker image",
	"Tag":       "Tag is which tag of the Docker image is being referenced",
	"ID":        "ID is the identifier for the Docker image",
}

func (DockerImageReference) SwaggerDoc() map[string]string {
	return map_DockerImageReference
}

var map_Image = map[string]string{
	"":                           "Image is an immutable representation of a Docker image and metadata at a point in time.",
	"metadata":                   "Standard object's metadata.",
	"dockerImageReference":       "DockerImageReference is the string that can be used to pull this image.",
	"dockerImageMetadata":        "DockerImageMetadata contains metadata about this image",
	"dockerImageMetadataVersion": "DockerImageMetadataVersion conveys the version of the object, which if empty defaults to \"1.0\"",
	"dockerImageManifest":        "DockerImageManifest is the raw JSON of the manifest",
	"dockerImageLayers":          "DockerImageLayers represents the layers in the image. May not be set if the image does not define that data.",
	"dockerConfigImage":          "DockerConfigImage is a JSON blob that the runtime uses to set up the container.",
}

func (Image) SwaggerDoc() map[string]string {
	return map_Image
}

var map_ImageImportSpec = map[string]string{
	"":                "ImageImportSpec describes a request to import a specific image.",
	"from":            "From is the source of an image to import; only kind DockerImage is allowed",
	"to":              "To is a tag in the current image stream to assign the imported image to, if name is not specified the default tag from from.name will be used",
	"importPolicy":    "ImportPolicy is the policy controlling how the image is imported",
	"includeManifest": "IncludeManifest determines if the manifest for each image is returned in the response",
}

func (ImageImportSpec) SwaggerDoc() map[string]string {
	return map_ImageImportSpec
}

var map_ImageImportStatus = map[string]string{
	"":       "ImageImportStatus describes the result of an image import.",
	"status": "Status is the status of the image import, including errors encountered while retrieving the image",
	"image":  "Image is the metadata of that image, if the image was located",
	"tag":    "Tag is the tag this image was located under, if any",
}

func (ImageImportStatus) SwaggerDoc() map[string]string {
	return map_ImageImportStatus
}

var map_ImageLayer = map[string]string{
	"":          "ImageLayer represents a single layer of the image. Some images may have multiple layers. Some may have none.",
	"name":      "Name of the layer as defined by the underlying store.",
	"size":      "Size of the layer as defined by the underlying store.",
	"MediaType": "MediaType of the referenced object.",
}

func (ImageLayer) SwaggerDoc() map[string]string {
	return map_ImageLayer
}

var map_ImageList = map[string]string{
	"":         "ImageList is a list of Image objects.",
	"metadata": "Standard object's metadata.",
	"items":    "Items is a list of images",
}

func (ImageList) SwaggerDoc() map[string]string {
	return map_ImageList
}

var map_ImageStream = map[string]string{
	"":         "ImageStream stores a mapping of tags to images, metadata overrides that are applied when images are tagged in a stream, and an optional reference to a Docker image repository on a registry.",
	"metadata": "Standard object's metadata.",
	"spec":     "Spec describes the desired state of this stream",
	"status":   "Status describes the current state of this stream",
}

func (ImageStream) SwaggerDoc() map[string]string {
	return map_ImageStream
}

var map_ImageStreamImage = map[string]string{
	"":         "ImageStreamImage represents an Image that is retrieved by image name from an ImageStream.",
	"metadata": "Standard object's metadata.",
	"image":    "Image associated with the ImageStream and image name.",
}

func (ImageStreamImage) SwaggerDoc() map[string]string {
	return map_ImageStreamImage
}

var map_ImageStreamImport = map[string]string{
	"":         "ImageStreamImport imports an image from remote repositories into OpenShift.",
	"metadata": "Standard object's metadata.",
	"spec":     "Spec is a description of the images that the user wishes to import",
	"status":   "Status is the the result of importing the image",
}

func (ImageStreamImport) SwaggerDoc() map[string]string {
	return map_ImageStreamImport
}

var map_ImageStreamImportSpec = map[string]string{
	"":           "ImageStreamImportSpec defines what images should be imported.",
	"import":     "Import indicates whether to perform an import - if so, the specified tags are set on the spec and status of the image stream defined by the type meta.",
	"repository": "Repository is an optional import of an entire Docker image repository. A maximum limit on the number of tags imported this way is imposed by the server.",
	"images":     "Images are a list of individual images to import.",
}

func (ImageStreamImportSpec) SwaggerDoc() map[string]string {
	return map_ImageStreamImportSpec
}

var map_ImageStreamImportStatus = map[string]string{
	"":           "ImageStreamImportStatus contains information about the status of an image stream import.",
	"import":     "Import is the image stream that was successfully updated or created when 'to' was set.",
	"repository": "Repository is set if spec.repository was set to the outcome of the import",
	"images":     "Images is set with the result of importing spec.images",
}

func (ImageStreamImportStatus) SwaggerDoc() map[string]string {
	return map_ImageStreamImportStatus
}

var map_ImageStreamList = map[string]string{
	"":         "ImageStreamList is a list of ImageStream objects.",
	"metadata": "Standard object's metadata.",
	"items":    "Items is a list of imageStreams",
}

func (ImageStreamList) SwaggerDoc() map[string]string {
	return map_ImageStreamList
}

var map_ImageStreamMapping = map[string]string{
	"":         "ImageStreamMapping represents a mapping from a single tag to a Docker image as well as the reference to the Docker image stream the image came from.",
	"metadata": "Standard object's metadata.",
	"image":    "Image is a Docker image.",
	"tag":      "Tag is a string value this image can be located with inside the stream.",
}

func (ImageStreamMapping) SwaggerDoc() map[string]string {
	return map_ImageStreamMapping
}

var map_ImageStreamSpec = map[string]string{
	"": "ImageStreamSpec represents options for ImageStreams.",
	"dockerImageRepository": "DockerImageRepository is optional, if specified this stream is backed by a Docker repository on this server",
	"tags":                  "Tags map arbitrary string values to specific image locators",
}

func (ImageStreamSpec) SwaggerDoc() map[string]string {
	return map_ImageStreamSpec
}

var map_ImageStreamStatus = map[string]string{
	"": "ImageStreamStatus contains information about the state of this image stream.",
	"dockerImageRepository": "DockerImageRepository represents the effective location this stream may be accessed at. May be empty until the server determines where the repository is located",
	"tags":                  "Tags are a historical record of images associated with each tag. The first entry in the TagEvent array is the currently tagged image.",
}

func (ImageStreamStatus) SwaggerDoc() map[string]string {
	return map_ImageStreamStatus
}

var map_ImageStreamTag = map[string]string{
	"":           "ImageStreamTag represents an Image that is retrieved by tag name from an ImageStream.",
	"metadata":   "Standard object's metadata.",
	"tag":        "Tag is the spec tag associated with this image stream tag, and it may be null if only pushes have occured to this image stream.",
	"generation": "Generation is the current generation of the tagged image - if tag is provided and this value is not equal to the tag generation, a user has requested an import that has not completed, or Conditions will be filled out indicating any error.",
	"conditions": "Conditions is an array of conditions that apply to the image stream tag.",
	"image":      "Image associated with the ImageStream and tag.",
}

func (ImageStreamTag) SwaggerDoc() map[string]string {
	return map_ImageStreamTag
}

var map_ImageStreamTagList = map[string]string{
	"":         "ImageStreamTagList is a list of ImageStreamTag objects.",
	"metadata": "Standard object's metadata.",
	"items":    "Items is the list of image stream tags",
}

func (ImageStreamTagList) SwaggerDoc() map[string]string {
	return map_ImageStreamTagList
}

var map_NamedTagEventList = map[string]string{
	"":           "NamedTagEventList relates a tag to its image history.",
	"tag":        "Tag is the tag for which the history is recorded",
	"items":      "Standard object's metadata.",
	"conditions": "Conditions is an array of conditions that apply to the tag event list.",
}

func (NamedTagEventList) SwaggerDoc() map[string]string {
	return map_NamedTagEventList
}

var map_RepositoryImportSpec = map[string]string{
	"":                "RepositoryImportSpec describes a request to import images from a Docker image repository.",
	"from":            "From is the source for the image repository to import; only kind DockerImage and a name of a Docker image repository is allowed",
	"importPolicy":    "ImportPolicy is the policy controlling how the image is imported",
	"includeManifest": "IncludeManifest determines if the manifest for each image is returned in the response",
}

func (RepositoryImportSpec) SwaggerDoc() map[string]string {
	return map_RepositoryImportSpec
}

var map_RepositoryImportStatus = map[string]string{
	"":               "RepositoryImportStatus describes the result of an image repository import",
	"status":         "Status reflects whether any failure occurred during import",
	"images":         "Images is a list of images successfully retrieved by the import of the repository.",
	"additionalTags": "AdditionalTags are tags that exist in the repository but were not imported because a maximum limit of automatic imports was applied.",
}

func (RepositoryImportStatus) SwaggerDoc() map[string]string {
	return map_RepositoryImportStatus
}

var map_TagEvent = map[string]string{
	"":                     "TagEvent is used by ImageStreamStatus to keep a historical record of images associated with a tag.",
	"created":              "Created holds the time the TagEvent was created",
	"dockerImageReference": "DockerImageReference is the string that can be used to pull this image",
	"image":                "Image is the image",
	"generation":           "Generation is the spec tag generation that resulted in this tag being updated",
}

func (TagEvent) SwaggerDoc() map[string]string {
	return map_TagEvent
}

var map_TagEventCondition = map[string]string{
	"":                   "TagEventCondition contains condition information for a tag event.",
	"type":               "Type of tag event condition, currently only ImportSuccess",
	"status":             "Status of the condition, one of True, False, Unknown.",
	"lastTransitionTime": "LastTransitionTIme is the time the condition transitioned from one status to another.",
	"reason":             "Reason is a brief machine readable explanation for the condition's last transition.",
	"message":            "Message is a human readable description of the details about last transition, complementing reason.",
	"generation":         "Generation is the spec tag generation that this status corresponds to",
}

func (TagEventCondition) SwaggerDoc() map[string]string {
	return map_TagEventCondition
}

var map_TagImportPolicy = map[string]string{
	"":          "TagImportPolicy describes the tag import policy",
	"insecure":  "Insecure is true if the server may bypass certificate verification or connect directly over HTTP during image import.",
	"scheduled": "Scheduled indicates to the server that this tag should be periodically checked to ensure it is up to date, and imported",
}

func (TagImportPolicy) SwaggerDoc() map[string]string {
	return map_TagImportPolicy
}

var map_TagReference = map[string]string{
	"":             "TagReference specifies optional annotations for images using this tag and an optional reference to an ImageStreamTag, ImageStreamImage, or DockerImage this tag should track.",
	"name":         "Name of the tag",
	"annotations":  "Annotations associated with images using this tag",
	"from":         "From is a reference to an image stream tag or image stream this tag should track",
	"reference":    "Reference states if the tag will be imported. Default value is false, which means the tag will be imported.",
	"generation":   "Generation is the image stream generation that updated this tag - setting it to 0 is an indication that the generation must be updated. Legacy clients will send this as nil, which means the client doesn't know or care.",
	"importPolicy": "Import is information that controls how images may be imported by the server.",
}

func (TagReference) SwaggerDoc() map[string]string {
	return map_TagReference
}
