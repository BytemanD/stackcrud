package cli

import (
	"os"
	"reflect"

	"github.com/jedib0t/go-pretty/v6/table"
	"github.com/jedib0t/go-pretty/v6/text"
)

// TODO: move to easygo

type Field struct {
	Name string
	Text string
}

type DataTable struct {
	ShortFields []Field
	LongFields  []Field
	Item        interface{}
	Slots       map[string]func(item interface{}) interface{}
	Title       string
}

func (dataTable DataTable) Print(long bool) {
	tableWriter := table.NewWriter()
	tableWriter.Style().Format.Header = text.FormatDefault
	tableWriter.SetOutputMirror(os.Stdout)

	headerRow := table.Row{"Field", "Value"}
	fields := dataTable.ShortFields
	if long {
		fields = append(fields, dataTable.LongFields...)
	}
	tableWriter.AppendHeader(headerRow)
	reflectValue := reflect.ValueOf(dataTable.Item)
	for _, field := range fields {
		var (
			fieldValue interface{}
			fieldLabel string
		)
		if field.Text == "" {
			fieldLabel = field.Name
		} else {
			fieldLabel = field.Text
		}
		if _, ok := dataTable.Slots[field.Name]; ok {
			fieldValue = dataTable.Slots[field.Name](dataTable.Item)
		} else {
			fieldValue = reflectValue.FieldByName(field.Name)
		}
		tableWriter.AppendRow(table.Row{fieldLabel, fieldValue})
	}
	if dataTable.Title != "" {
		tableWriter.SetTitle(dataTable.Title)
		tableWriter.Style().Title.Align = text.AlignCenter
	}
	tableWriter.Render()
}

type DataListTable struct {
	ShortHeaders  []string
	LongHeaders   []string
	HeaderLabel   map[string]string
	Items         []interface{}
	SortBy        []table.SortBy
	ColumnConfigs []table.ColumnConfig
	Slots         map[string]func(item interface{}) interface{}
	Title         string
}

func (dataTable DataListTable) Print(long bool) {
	tableWriter := table.NewWriter()
	tableWriter.Style().Format.Header = text.FormatDefault
	tableWriter.SetOutputMirror(os.Stdout)

	headerRow := table.Row{}
	titles := dataTable.ShortHeaders
	if long {
		titles = append(titles, dataTable.LongHeaders...)
	}
	for _, header := range titles {
		var title string
		if _, ok := dataTable.HeaderLabel[header]; ok {
			title = dataTable.HeaderLabel[header]
		} else {
			title = header
		}
		headerRow = append(headerRow, title)
	}
	tableWriter.AppendHeader(headerRow)

	for _, item := range dataTable.Items {
		reflectValue := reflect.ValueOf(item)
		row := table.Row{}
		for _, name := range titles {
			var value interface{}
			if _, ok := dataTable.Slots[name]; ok {
				value = dataTable.Slots[name](item)
			} else {
				value = reflectValue.FieldByName(name)
			}
			row = append(row, value)
		}
		tableWriter.AppendRow(row)
	}
	if dataTable.Title != "" {
		tableWriter.SetTitle(dataTable.Title)
		tableWriter.Style().Title.Align = text.AlignCenter
	}
	tableWriter.SortBy(dataTable.SortBy)
	tableWriter.SetColumnConfigs(dataTable.ColumnConfigs)
	tableWriter.Render()
}