package yaml

import (
	"io"

	"k8s.io/apimachinery/pkg/apis/meta/v1/unstructured"
	"k8s.io/apimachinery/pkg/runtime/serializer/json"
	"k8s.io/apimachinery/pkg/runtime/serializer/streaming"
	"k8s.io/apimachinery/pkg/runtime/serializer/yaml"
)

func ParseUnstructuredAll(r io.ReadCloser) ([]*unstructured.Unstructured, error) {
	yamlDecoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)
	frameReader := json.YAMLFramer.NewFrameReader(r)

	decoder := streaming.NewDecoder(frameReader, yamlDecoder)
	var objs []*unstructured.Unstructured
	for {
		obj := &unstructured.Unstructured{}
		_, _, err := decoder.Decode(nil, obj)

		if err != nil {
			if err == io.EOF {
				break
			}

			return nil, err
		}

		objs = append(objs, obj)
	}

	return objs, nil
}

func ParseUnstructured(b []byte) (*unstructured.Unstructured, error) {
	yamlDecoder := yaml.NewDecodingSerializer(unstructured.UnstructuredJSONScheme)

	into := &unstructured.Unstructured{}
	_, _, err := yamlDecoder.Decode(b, nil, into)

	if err != nil {
		return nil, err
	}

	return into, nil
}
