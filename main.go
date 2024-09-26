package main

import (
	"encoding/binary"
	"fmt"
	"log"
	"net/http"
	"os"
	"strconv"
	"sync"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/goburrow/modbus"
	"github.com/joho/godotenv"
)

type ModbusData struct {
	modbus1Data []int64
	modbus2Data []int64
	mux         sync.Mutex
}

func main() {

	log.Println("Starting the application...")

	err := godotenv.Load()
	if err != nil {
		log.Fatalf("Error loading .env file: %v", err)
	}

	appPort := os.Getenv("APP_PORT")

	modbusData := &ModbusData{}

	go backgroundTask(modbusData)

	router := gin.Default()

	router.GET("/healthcheck", func(c *gin.Context) {
		c.JSON(http.StatusOK, gin.H{
			"status": "Online.",
		})
	})

	router.GET("/modbus-data", getModbusDataHandler(modbusData))

	router.Run(fmt.Sprintf("0.0.0.0:%s", appPort))
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

func backgroundTask(modbusData *ModbusData) {

	machine1IP := os.Getenv("MACHINE_1_IP")
	machine1Port := os.Getenv("MACHINE_1_PORT")
	machine1Start, err := strconv.Atoi(os.Getenv("MACHINE_1_START_ADDRESS"))
	if err != nil {
		log.Printf("Cannot parse \"machine1Start\" to int")
		return
	}
	machine1Count, err := strconv.Atoi(os.Getenv("MACHINE_1_READ_COUNT"))
	if err != nil {
		log.Printf("Cannot parse \"machine1Count\" to int")
		return
	}

	machine2IP := os.Getenv("MACHINE_2_IP")
	machine2Port := os.Getenv("MACHINE_2_PORT")
	machine2Start, err := strconv.Atoi(os.Getenv("MACHINE_2_START_ADDRESS"))
	if err != nil {
		log.Printf("Cannot parse \"machine2Start\" to int")
		return
	}
	machine2Count, err := strconv.Atoi(os.Getenv("MACHINE_2_READ_COUNT"))
	if err != nil {
		log.Printf("Cannot parse \"machine2Count\" to int")
		return
	}

	handler1, err1 := createModbusHandler(machine1IP, machine1Port)
	if err1 != nil {
		log.Printf("Error creating handler for machine 1: %v", err1)
		return
	}
	defer handler1.Close()

	handler2, err2 := createModbusHandler(machine2IP, machine2Port)
	if err2 != nil {
		log.Printf("Error creating handler for machine 2: %v", err2)
		return
	}
	defer handler2.Close()

	client1 := modbus.NewClient(handler1)
	client2 := modbus.NewClient(handler2)

	isUseModbus, err := strconv.ParseBool(os.Getenv("USE_MODSIM"))
	if err != nil {
		log.Printf("Error convert isUseModbus to bool")
		return
	}

	if isUseModbus {
		machine1Start -= 1
		machine2Start -= 1
	}

	for {

		data1, err1 := readModbusData(client1, machine1Start, machine1Count)
		if err1 != nil {
			log.Printf("Error reading from machine 1: %v", err1)
		}

		data2, err2 := readModbusData(client2, machine2Start, machine2Count)
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
func createModbusHandler(ip, port string) (*modbus.TCPClientHandler, error) {
	handler := modbus.NewTCPClientHandler(fmt.Sprintf("%s:%s", ip, port))
	handler.Timeout = 10 * time.Second
	handler.SlaveId = 1

	err := handler.Connect()
	if err != nil {
		return nil, fmt.Errorf("failed to connect TCPClient")
	}

	return handler, nil
}

func readModbusData(client modbus.Client, start, count int) ([]int64, error) {
	results, err := client.ReadHoldingRegisters(99, 10)
	if err != nil {
		return nil, fmt.Errorf("failed to read registers")
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
