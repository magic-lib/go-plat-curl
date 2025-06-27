package curl

import (
	"bytes"
	"fmt"
	"github.com/magic-lib/go-plat-utils/conv"
	"io"
	"mime/multipart"
	"net/http"
	"net/textproto"
)

// UploadFile 上传文件结构
type UploadFile struct {
	FieldName       string `json:"field_name"`        // 表单字段名
	FileName        string `json:"file_name"`         // 文件名
	FileContentType string `json:"file_content_type"` // 文件类型
	FileData        []byte `json:"file_data"`         // 文件数据
}

// getFileBody headers
func (g *genRequest) getFileBody() (bool, *bytes.Buffer, http.Header, error) {
	if g.Data == nil {
		return false, nil, nil, nil
	}
	dataMap, ok := g.Data.(map[string]any)
	if !ok {
		return false, nil, nil, nil
	}
	fileList := make([]*UploadFile, 0)

	otherFields := make(map[string]string)
	for k, v := range dataMap {
		if uf, ok := v.(*UploadFile); ok {
			uf.FieldName = k
			fileList = append(fileList, uf)
			continue
		}
		otherFields[k] = conv.String(v)
	}
	if len(fileList) == 0 {
		return false, nil, nil, nil
	}

	body, header, err := uploadMultipleFiles(fileList, otherFields)
	return true, body, header, err
}

func hasFileData(data any) bool {
	if data == nil {
		return false
	}
	dataMap, ok := data.(map[string]any)
	if !ok {
		return false
	}
	for _, v := range dataMap {
		if _, ok := v.(*UploadFile); ok {
			return true
		}
	}
	return false
}

// 上传多个文件并携带额外表单字段
func uploadMultipleFiles(files []*UploadFile, fields map[string]string) (*bytes.Buffer, http.Header, error) {
	// 创建一个临时的buffer用于构建multipart表单
	body := &bytes.Buffer{}
	writer := multipart.NewWriter(body)

	// 添加额外的表单字段
	for key, value := range fields {
		if err := writer.WriteField(key, value); err != nil {
			return nil, nil, fmt.Errorf("写入表单字段失败: %v", err)
		}
	}

	// 添加多个文件
	for _, file := range files {
		// 创建一个文件字段
		fileField, err := writer.CreateFormFile(file.FieldName, file.FileName)
		if err != nil {
			return nil, nil, fmt.Errorf("创建表单文件失败: %v", err)
		}

		// 设置Content-Type头（如果提供）
		if file.FileContentType != "" {
			// 创建带指定Header的文件字段
			w, err := file.createFormFile(writer)
			if err != nil {
				return nil, nil, fmt.Errorf("创建表单文件失败: %v", err)
			}
			// 将文件数据写入
			if _, err := io.Copy(w, bytes.NewReader(file.FileData)); err != nil {
				return nil, nil, fmt.Errorf("写入文件数据失败: %v", err)
			}
			continue
		}

		// 将图片数据写入文件字段
		_, err = io.Copy(fileField, bytes.NewReader(file.FileData))
		if err != nil {
			return nil, nil, fmt.Errorf("写入文件数据失败: %v", err)
		}
	}

	// 关闭writer以完成表单构建
	if err := writer.Close(); err != nil {
		return nil, nil, fmt.Errorf("关闭表单写入器失败: %v", err)
	}

	return body, http.Header{
		headerContentType: []string{writer.FormDataContentType()},
	}, nil
}

func (f *UploadFile) createFormFile(w *multipart.Writer) (io.Writer, error) {
	h := make(textproto.MIMEHeader)
	h.Set("Content-Disposition",
		fmt.Sprintf(`form-data; name="%s"; filename="%s"`,
			f.FieldName, f.FileName))
	h.Set(headerContentType, f.FileContentType)
	return w.CreatePart(h)
}
