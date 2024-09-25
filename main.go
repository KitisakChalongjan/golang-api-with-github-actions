package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
)

type ModbusData struct {
	modbus1Data []int64
	modbus2Data []int64
	mux         sync.Mutex
}

func main() {

	log.Println("Starting the application...")

	machine1IP := "192.168.1.33"
	machine1Port := "501"

	machine2IP := "192.168.1.33"
	machine2Port := "502"

	modbusData := &ModbusData{}

	go backgroundTask(modbusData, machine1IP, machine1Port, machine2IP, machine2Port)

	router := gin.Default()

	router.GET("/healthcheck", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "Online.",
		})
	})

	router.GET("/modbus-data", getModbusDataHandler(modbusData))

	router.Run("0.0.0.0:8000")
}

func getModbusDataHandler(modbusData *ModbusData) gin.HandlerFunc {
	return func(c *gin.Context) {

		modbusData.mux.Lock()
		data1 := modbusData.modbus1Data
		data2 := modbusData.modbus2Data
		modbusData.mux.Unlock()

		if data1 == nil || data2 == nil {
			c.JSON(http.StatusInternalServerError, gin.H{"error": "Data not available"})
			return
		}

		c.JSON(http.StatusOK, gin.H{
			"modbus1_data": fmt.Sprintf("%d", data1),
			"modbus2_data": fmt.Sprintf("%d", data2),
		})
	}
}

func backgroundTask(modbusData *ModbusData, machine1IP, machine1Port, machine2IP, machine2Port string) {
	for {

		data1, err1 := readModbusData(machine1IP, machine1Port)
		if err1 != nil {
			log.Printf("Error reading from machine 1: %v", err1)
		}

		data2, err2 := readModbusData(machine2IP, machine2Port)
		if err2 != nil {
			log.Printf("Error reading from machine 2: %v", err2)
		}

		fmt.Printf("Data1 : %d\nData2 : %d\n", data1, data2)

		modbusData.mux.Lock()
		modbusData.modbus1Data = data1
		modbusData.modbus2Data = data2
		modbusData.mux.Unlock()

		time.Sleep(5 * time.Second)
	}
}

func readModbusData(ip string, port string) ([]int64, error) {
	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%s", ip, port))
	handler.Timeout = 10 * time.Second
	handler.SlaveId = 1

	err := handler.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect to %s: %v", ip, err)
	}
	defer handler.Close()

	client := modbus.NewClient(handler)
	results, err := client.ReadHoldingRegisters(99, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to read registers from %s: %v", ip, err)
	}

	intResults := convertByteToInt(results)

	return intResults, nil
}

func convertByteToInt(bytes []byte) []int64 {
	if len(bytes)%2 != 0 {
		fmt.Println("Byte slice length is not even, cannot convert to 16-bit integers.")
		return nil
	}

	intValues := make([]int64, len(bytes)/2)

	for i := 0; i < len(bytes); i += 2 {
		value := binary.BigEndian.Uint16(bytes[i : i+2])
		intValues[i/2] = int64(value)
	}

	return intValues
}
