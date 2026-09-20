package main

import (
	"bufio"
	"encoding/base64"
	"fmt"
	"os"
	"planning-reparti/internal/comm"
	netstate "planning-reparti/internal/net"
	"planning-reparti/internal/utils"
)

//FICHIERH CTL BRIDGE

func readCtlInput(st *netstate.State, logger *utils.Logger, path string) {
	file, err := os.OpenFile(path, os.O_RDONLY, os.ModeNamedPipe)
	if err != nil {
		logger.Info("ctl", "erreur ouverture ctl-in : "+err.Error())
		return
	}
	defer file.Close()

	scanner := bufio.NewScanner(file)

	for scanner.Scan() {
		line := scanner.Text()

		logger.Info("ctl", "message reçu de CTL : "+line)

		to := comm.FindVal(line, "to")
		if to == "" {
			to = st.Successor
		}

		payload := base64.RawStdEncoding.EncodeToString([]byte(line))

		msg := comm.Build(map[string]string{
			"type":    "CTL_MSG",
			"site":    st.Name,
			"to":      to,
			"payload": payload,
		})

		logger.Info("ctl", "message transmis au réseau : "+msg)
		fmt.Fprintln(os.Stdout, msg)
	}
}

func handleCtlMsg(st *netstate.State, logger *utils.Logger, line string, ctlOut string) {
	to := comm.FindVal(line, "to")

	if to != st.Name && to != "ALL" {
		return
	}
	payload := comm.FindVal(line, "payload")
	decoded, err := base64.RawStdEncoding.DecodeString(payload)
	if err != nil {
		logger.Info("ctl", "erreur décodage payload : "+err.Error())
		return
	}

	body := string(decoded)
	logger.Info("ctl", "message réseau reçu pour CTL : "+body)
	innerType := comm.FindVal(body, "type")

	if ctlOut == "" {
		logger.Info("ctl", "ctl-out non configuré")
		return
	}
	//	net traite ces 2 msgs puis send to ctl
	if innerType == "SHOW_NET_STATE" {
		netState := buildNetStateToCtl(st)
		logger.Info("ctl", "NET_STATE construit : "+netState)
		logger.Info("ctl", "envoi NET_STATE vers CTL via "+st.CtlOut)
		sendToCtl(st.CtlOut, netState)
		return
	}

	if innerType == "LEAVE_REQUEST" {
		handleLeaveRequest(st, logger, body)
		return
	}

	file, err := os.OpenFile(ctlOut, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		logger.Info("ctl", "erreur écriture ctl-out : "+err.Error())
		return
	}
	defer file.Close()

	fmt.Fprintln(file, body)
}

func sendToCtl(path string, msg string) {
	file, err := os.OpenFile(path, os.O_WRONLY, os.ModeNamedPipe)
	if err != nil {
		fmt.Fprintln(os.Stderr, "sendToCtl erreur ouverture "+path+" : "+err.Error())
		return
	}
	defer file.Close()

	fmt.Fprintln(file, msg)
}
