// +build !ignore_autogenerated

// This file was autogenerated by deepcopy-gen. Do not edit it manually!

package v1

import (
	api "k8s.io/kubernetes/pkg/api"
	unversioned "k8s.io/kubernetes/pkg/api/unversioned"
	api_v1 "k8s.io/kubernetes/pkg/api/v1"
	conversion "k8s.io/kubernetes/pkg/conversion"
	runtime "k8s.io/kubernetes/pkg/runtime"
)

func init() {
	if err := api.Scheme.AddGeneratedDeepCopyFuncs(
		DeepCopy_v1_DockerImageReference,
		DeepCopy_v1_Image,
		DeepCopy_v1_ImageImportSpec,
		DeepCopy_v1_ImageImportStatus,
		DeepCopy_v1_ImageLayer,
		DeepCopy_v1_ImageList,
		DeepCopy_v1_ImageStream,
		DeepCopy_v1_ImageStreamImage,
		DeepCopy_v1_ImageStreamImport,
		DeepCopy_v1_ImageStreamImportSpec,
		DeepCopy_v1_ImageStreamImportStatus,
		DeepCopy_v1_ImageStreamList,
		DeepCopy_v1_ImageStreamMapping,
		DeepCopy_v1_ImageStreamSpec,
		DeepCopy_v1_ImageStreamStatus,
		DeepCopy_v1_ImageStreamTag,
		DeepCopy_v1_ImageStreamTagList,
		DeepCopy_v1_NamedTagEventList,
		DeepCopy_v1_RepositoryImportSpec,
		DeepCopy_v1_RepositoryImportStatus,
		DeepCopy_v1_TagEvent,
		DeepCopy_v1_TagEventCondition,
		DeepCopy_v1_TagImportPolicy,
		DeepCopy_v1_TagReference,
	); err != nil {
		// if one of the deep copy functions is malformed, detect it immediately.
		panic(err)
	}
}

func DeepCopy_v1_DockerImageReference(in DockerImageReference, out *DockerImageReference, c *conversion.Cloner) error {
	out.Registry = in.Registry
	out.Namespace = in.Namespace
	out.Name = in.Name
	out.Tag = in.Tag
	out.ID = in.ID
	return nil
}

func DeepCopy_v1_Image(in Image, out *Image, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_TypeMeta(in.TypeMeta, &out.TypeMeta, c); err != nil {
		return err
	}
	if err := api_v1.DeepCopy_v1_ObjectMeta(in.ObjectMeta, &out.ObjectMeta, c); err != nil {
		return err
	}
	out.DockerImageReference = in.DockerImageReference
	if err := runtime.DeepCopy_runtime_RawExtension(in.DockerImageMetadata, &out.DockerImageMetadata, c); err != nil {
		return err
	}
	out.DockerImageMetadataVersion = in.DockerImageMetadataVersion
	out.DockerImageManifest = in.DockerImageManifest
	if in.DockerImageLayers != nil {
		in, out := in.DockerImageLayers, &out.DockerImageLayers
		*out = make([]ImageLayer, len(in))
		for i := range in {
			if err := DeepCopy_v1_ImageLayer(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.DockerImageLayers = nil
	}
	out.DockerConfigImage = in.DockerConfigImage
	return nil
}

func DeepCopy_v1_ImageImportSpec(in ImageImportSpec, out *ImageImportSpec, c *conversion.Cloner) error {
	if err := api_v1.DeepCopy_v1_ObjectReference(in.From, &out.From, c); err != nil {
		return err
	}
	if in.To != nil {
		in, out := in.To, &out.To
		*out = new(api_v1.LocalObjectReference)
		if err := api_v1.DeepCopy_v1_LocalObjectReference(*in, *out, c); err != nil {
			return err
		}
	} else {
		out.To = nil
	}
	if err := DeepCopy_v1_TagImportPolicy(in.ImportPolicy, &out.ImportPolicy, c); err != nil {
		return err
	}
	out.IncludeManifest = in.IncludeManifest
	return nil
}

func DeepCopy_v1_ImageImportStatus(in ImageImportStatus, out *ImageImportStatus, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_Status(in.Status, &out.Status, c); err != nil {
		return err
	}
	if in.Image != nil {
		in, out := in.Image, &out.Image
		*out = new(Image)
		if err := DeepCopy_v1_Image(*in, *out, c); err != nil {
			return err
		}
	} else {
		out.Image = nil
	}
	out.Tag = in.Tag
	return nil
}

func DeepCopy_v1_ImageLayer(in ImageLayer, out *ImageLayer, c *conversion.Cloner) error {
	out.Name = in.Name
	out.Size = in.Size
	out.MediaType = in.MediaType
	return nil
}

func DeepCopy_v1_ImageList(in ImageList, out *ImageList, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_TypeMeta(in.TypeMeta, &out.TypeMeta, c); err != nil {
		return err
	}
	if err := unversioned.DeepCopy_unversioned_ListMeta(in.ListMeta, &out.ListMeta, c); err != nil {
		return err
	}
	if in.Items != nil {
		in, out := in.Items, &out.Items
		*out = make([]Image, len(in))
		for i := range in {
			if err := DeepCopy_v1_Image(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

func DeepCopy_v1_ImageStream(in ImageStream, out *ImageStream, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_TypeMeta(in.TypeMeta, &out.TypeMeta, c); err != nil {
		return err
	}
	if err := api_v1.DeepCopy_v1_ObjectMeta(in.ObjectMeta, &out.ObjectMeta, c); err != nil {
		return err
	}
	if err := DeepCopy_v1_ImageStreamSpec(in.Spec, &out.Spec, c); err != nil {
		return err
	}
	if err := DeepCopy_v1_ImageStreamStatus(in.Status, &out.Status, c); err != nil {
		return err
	}
	return nil
}

func DeepCopy_v1_ImageStreamImage(in ImageStreamImage, out *ImageStreamImage, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_TypeMeta(in.TypeMeta, &out.TypeMeta, c); err != nil {
		return err
	}
	if err := api_v1.DeepCopy_v1_ObjectMeta(in.ObjectMeta, &out.ObjectMeta, c); err != nil {
		return err
	}
	if err := DeepCopy_v1_Image(in.Image, &out.Image, c); err != nil {
		return err
	}
	return nil
}

func DeepCopy_v1_ImageStreamImport(in ImageStreamImport, out *ImageStreamImport, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_TypeMeta(in.TypeMeta, &out.TypeMeta, c); err != nil {
		return err
	}
	if err := api_v1.DeepCopy_v1_ObjectMeta(in.ObjectMeta, &out.ObjectMeta, c); err != nil {
		return err
	}
	if err := DeepCopy_v1_ImageStreamImportSpec(in.Spec, &out.Spec, c); err != nil {
		return err
	}
	if err := DeepCopy_v1_ImageStreamImportStatus(in.Status, &out.Status, c); err != nil {
		return err
	}
	return nil
}

func DeepCopy_v1_ImageStreamImportSpec(in ImageStreamImportSpec, out *ImageStreamImportSpec, c *conversion.Cloner) error {
	out.Import = in.Import
	if in.Repository != nil {
		in, out := in.Repository, &out.Repository
		*out = new(RepositoryImportSpec)
		if err := DeepCopy_v1_RepositoryImportSpec(*in, *out, c); err != nil {
			return err
		}
	} else {
		out.Repository = nil
	}
	if in.Images != nil {
		in, out := in.Images, &out.Images
		*out = make([]ImageImportSpec, len(in))
		for i := range in {
			if err := DeepCopy_v1_ImageImportSpec(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Images = nil
	}
	return nil
}

func DeepCopy_v1_ImageStreamImportStatus(in ImageStreamImportStatus, out *ImageStreamImportStatus, c *conversion.Cloner) error {
	if in.Import != nil {
		in, out := in.Import, &out.Import
		*out = new(ImageStream)
		if err := DeepCopy_v1_ImageStream(*in, *out, c); err != nil {
			return err
		}
	} else {
		out.Import = nil
	}
	if in.Repository != nil {
		in, out := in.Repository, &out.Repository
		*out = new(RepositoryImportStatus)
		if err := DeepCopy_v1_RepositoryImportStatus(*in, *out, c); err != nil {
			return err
		}
	} else {
		out.Repository = nil
	}
	if in.Images != nil {
		in, out := in.Images, &out.Images
		*out = make([]ImageImportStatus, len(in))
		for i := range in {
			if err := DeepCopy_v1_ImageImportStatus(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Images = nil
	}
	return nil
}

func DeepCopy_v1_ImageStreamList(in ImageStreamList, out *ImageStreamList, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_TypeMeta(in.TypeMeta, &out.TypeMeta, c); err != nil {
		return err
	}
	if err := unversioned.DeepCopy_unversioned_ListMeta(in.ListMeta, &out.ListMeta, c); err != nil {
		return err
	}
	if in.Items != nil {
		in, out := in.Items, &out.Items
		*out = make([]ImageStream, len(in))
		for i := range in {
			if err := DeepCopy_v1_ImageStream(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

func DeepCopy_v1_ImageStreamMapping(in ImageStreamMapping, out *ImageStreamMapping, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_TypeMeta(in.TypeMeta, &out.TypeMeta, c); err != nil {
		return err
	}
	if err := api_v1.DeepCopy_v1_ObjectMeta(in.ObjectMeta, &out.ObjectMeta, c); err != nil {
		return err
	}
	if err := DeepCopy_v1_Image(in.Image, &out.Image, c); err != nil {
		return err
	}
	out.Tag = in.Tag
	return nil
}

func DeepCopy_v1_ImageStreamSpec(in ImageStreamSpec, out *ImageStreamSpec, c *conversion.Cloner) error {
	out.DockerImageRepository = in.DockerImageRepository
	if in.Tags != nil {
		in, out := in.Tags, &out.Tags
		*out = make([]TagReference, len(in))
		for i := range in {
			if err := DeepCopy_v1_TagReference(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Tags = nil
	}
	return nil
}

func DeepCopy_v1_ImageStreamStatus(in ImageStreamStatus, out *ImageStreamStatus, c *conversion.Cloner) error {
	out.DockerImageRepository = in.DockerImageRepository
	if in.Tags != nil {
		in, out := in.Tags, &out.Tags
		*out = make([]NamedTagEventList, len(in))
		for i := range in {
			if err := DeepCopy_v1_NamedTagEventList(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Tags = nil
	}
	return nil
}

func DeepCopy_v1_ImageStreamTag(in ImageStreamTag, out *ImageStreamTag, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_TypeMeta(in.TypeMeta, &out.TypeMeta, c); err != nil {
		return err
	}
	if err := api_v1.DeepCopy_v1_ObjectMeta(in.ObjectMeta, &out.ObjectMeta, c); err != nil {
		return err
	}
	if in.Tag != nil {
		in, out := in.Tag, &out.Tag
		*out = new(TagReference)
		if err := DeepCopy_v1_TagReference(*in, *out, c); err != nil {
			return err
		}
	} else {
		out.Tag = nil
	}
	out.Generation = in.Generation
	if in.Conditions != nil {
		in, out := in.Conditions, &out.Conditions
		*out = make([]TagEventCondition, len(in))
		for i := range in {
			if err := DeepCopy_v1_TagEventCondition(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Conditions = nil
	}
	if err := DeepCopy_v1_Image(in.Image, &out.Image, c); err != nil {
		return err
	}
	return nil
}

func DeepCopy_v1_ImageStreamTagList(in ImageStreamTagList, out *ImageStreamTagList, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_TypeMeta(in.TypeMeta, &out.TypeMeta, c); err != nil {
		return err
	}
	if err := unversioned.DeepCopy_unversioned_ListMeta(in.ListMeta, &out.ListMeta, c); err != nil {
		return err
	}
	if in.Items != nil {
		in, out := in.Items, &out.Items
		*out = make([]ImageStreamTag, len(in))
		for i := range in {
			if err := DeepCopy_v1_ImageStreamTag(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	return nil
}

func DeepCopy_v1_NamedTagEventList(in NamedTagEventList, out *NamedTagEventList, c *conversion.Cloner) error {
	out.Tag = in.Tag
	if in.Items != nil {
		in, out := in.Items, &out.Items
		*out = make([]TagEvent, len(in))
		for i := range in {
			if err := DeepCopy_v1_TagEvent(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Items = nil
	}
	if in.Conditions != nil {
		in, out := in.Conditions, &out.Conditions
		*out = make([]TagEventCondition, len(in))
		for i := range in {
			if err := DeepCopy_v1_TagEventCondition(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Conditions = nil
	}
	return nil
}

func DeepCopy_v1_RepositoryImportSpec(in RepositoryImportSpec, out *RepositoryImportSpec, c *conversion.Cloner) error {
	if err := api_v1.DeepCopy_v1_ObjectReference(in.From, &out.From, c); err != nil {
		return err
	}
	if err := DeepCopy_v1_TagImportPolicy(in.ImportPolicy, &out.ImportPolicy, c); err != nil {
		return err
	}
	out.IncludeManifest = in.IncludeManifest
	return nil
}

func DeepCopy_v1_RepositoryImportStatus(in RepositoryImportStatus, out *RepositoryImportStatus, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_Status(in.Status, &out.Status, c); err != nil {
		return err
	}
	if in.Images != nil {
		in, out := in.Images, &out.Images
		*out = make([]ImageImportStatus, len(in))
		for i := range in {
			if err := DeepCopy_v1_ImageImportStatus(in[i], &(*out)[i], c); err != nil {
				return err
			}
		}
	} else {
		out.Images = nil
	}
	if in.AdditionalTags != nil {
		in, out := in.AdditionalTags, &out.AdditionalTags
		*out = make([]string, len(in))
		copy(*out, in)
	} else {
		out.AdditionalTags = nil
	}
	return nil
}

func DeepCopy_v1_TagEvent(in TagEvent, out *TagEvent, c *conversion.Cloner) error {
	if err := unversioned.DeepCopy_unversioned_Time(in.Created, &out.Created, c); err != nil {
		return err
	}
	out.DockerImageReference = in.DockerImageReference
	out.Image = in.Image
	out.Generation = in.Generation
	return nil
}

func DeepCopy_v1_TagEventCondition(in TagEventCondition, out *TagEventCondition, c *conversion.Cloner) error {
	out.Type = in.Type
	out.Status = in.Status
	if err := unversioned.DeepCopy_unversioned_Time(in.LastTransitionTime, &out.LastTransitionTime, c); err != nil {
		return err
	}
	out.Reason = in.Reason
	out.Message = in.Message
	out.Generation = in.Generation
	return nil
}

func DeepCopy_v1_TagImportPolicy(in TagImportPolicy, out *TagImportPolicy, c *conversion.Cloner) error {
	out.Insecure = in.Insecure
	out.Scheduled = in.Scheduled
	return nil
}

func DeepCopy_v1_TagReference(in TagReference, out *TagReference, c *conversion.Cloner) error {
	out.Name = in.Name
	if in.Annotations != nil {
		in, out := in.Annotations, &out.Annotations
		*out = make(map[string]string)
		for key, val := range in {
			(*out)[key] = val
		}
	} else {
		out.Annotations = nil
	}
	if in.From != nil {
		in, out := in.From, &out.From
		*out = new(api_v1.ObjectReference)
		if err := api_v1.DeepCopy_v1_ObjectReference(*in, *out, c); err != nil {
			return err
		}
	} else {
		out.From = nil
	}
	out.Reference = in.Reference
	if in.Generation != nil {
		in, out := in.Generation, &out.Generation
		*out = new(int64)
		**out = *in
	} else {
		out.Generation = nil
	}
	if err := DeepCopy_v1_TagImportPolicy(in.ImportPolicy, &out.ImportPolicy, c); err != nil {
		return err
	}
	return nil
}
