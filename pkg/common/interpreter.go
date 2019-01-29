package common

import (
	"fmt"
	"strings"
	"time"
)

type J1587Interpreter struct {
}

func (i *J1587Interpreter) Interpret(message *J1587Message) (string, error) {
	sb := new(strings.Builder)

	sb.WriteString(fmt.Sprintf("<--  [%s]    ", time.Now().Format("3:04:05 PM")))
	sb.WriteString(fmt.Sprintf("%v\n", message.Raw))

	sb.WriteString("\n")

	midType := i.getMidDefinition(message.Mid)
	sb.WriteString(fmt.Sprintf(";    MID %d : %s\n", message.Mid, midType))

	pidType := i.getPidDefinition(message.Pid)
	sb.WriteString(fmt.Sprintf(";    PID %d : %s\n", message.Pid, pidType))

	switch message.Pid {
	case 128:
		i.interpretComponentIdRequest(sb, message.Data)
		break
	}

	sb.WriteString("----\n")

	return sb.String(), nil
}

func (i *J1587Interpreter) interpretComponentIdRequest(sb *strings.Builder, message []byte) {
	pidInfo := i.getPidDefinition(int(message[0]))

	sb.WriteString(fmt.Sprintf(";    Requested Parameter %d:\n", message[0]))
	sb.WriteString(fmt.Sprintf(";      %s\n", pidInfo))

	rd := i.getMidDefinition(int(message[1]))
	sb.WriteString(fmt.Sprintf(";    Receiver MID: %d - %s\n", message[1], rd))
}

func (i *J1587Interpreter) getMidDefinition(mid int) string {
	d := "Unknown"

	switch mid {
	case 188:
		d = "Vehicle Logic Control Unit"
		break
	case 196:
		d = "Farebox"
		break
	}

	return d
}

func (i *J1587Interpreter) getPidDefinition(pid int) string {
	d := "Unknown"

	switch pid {
	case 128:
		d = "Component Identification Request"
	case 234:
		d = "Software Identification"
		break
	}

	return d
}
