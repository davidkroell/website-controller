package v1

import "k8s.io/apimachinery/pkg/runtime"

// DeepCopyInto copies all properties of this object into another object of the
// same type that is provided as a pointer.
func (in *WebSite) DeepCopyInto(out *WebSite) {
	out.TypeMeta = in.TypeMeta
	out.ObjectMeta = in.ObjectMeta
	out.Spec = WebSiteSpec{
		HtmlContent: in.Spec.HtmlContent,
		Hostname:    in.Spec.Hostname,
		NginxImage:  in.Spec.NginxImage,
	}
}

// DeepCopyObject returns a generically typed copy of an object
func (in *WebSite) DeepCopyObject() runtime.Object {
	out := WebSite{}
	in.DeepCopyInto(&out)

	return &out
}

// DeepCopyObject returns a generically typed copy of an object
func (in *WebSiteList) DeepCopyObject() runtime.Object {
	out := WebSiteList{}
	out.TypeMeta = in.TypeMeta
	out.ListMeta = in.ListMeta

	if in.Items != nil {
		out.Items = make([]WebSite, len(in.Items))
		for i := range in.Items {
			in.Items[i].DeepCopyInto(&out.Items[i])
		}
	}

	return &out
}
