package keylogger

import (
	"bytes"
	"encoding/binary"
	"errors"
	"fmt"
	"io/ioutil"
	"os"
	"strings"
	"syscall"
	"unsafe"

	"github.com/mateusmaaia/keylogo/linux/mapping"
)

// KeyLogger wrapper around file descriptior
type KeyLogger struct {
	fd *os.File
}

type devices []string

func (d *devices) hasDevice(str string) bool {
	for _, device := range *d {
		if strings.Contains(str, device) {
			return true
		}
	}

	return false
}

// use lowercase names for devices, as we turn the device input name to lower case
var restrictedDevices = devices{"mouse"}
var allowedDevices = devices{"keyboard", "logitech mx keys", "switch"}

// New creates a new keylogger for a device path
func New(devPath string) (*KeyLogger, error) {
	k := &KeyLogger{}
	if !k.IsRoot() {
		return nil, errors.New("Must be run as root")
	}
	fd, err := os.OpenFile(devPath, os.O_RDWR, os.ModeCharDevice)
	k.fd = fd
	return k, err
}

// FindKeyboardDevice by going through each device registered on OS
// Mostly it will contain keyword - keyboard
// Returns the file path which contains events
func FindKeyboardDevice() string {
	path := "/sys/class/input/event%d/device/name"
	resolved := "/dev/input/event%d"

	for i := 0; i < 255; i++ {
		buff, err := ioutil.ReadFile(fmt.Sprintf(path, i))
		if err != nil {
			continue
		}

		deviceName := strings.ToLower(string(buff))

		if restrictedDevices.hasDevice(deviceName) {
			continue
		} else if allowedDevices.hasDevice(deviceName) {
			return fmt.Sprintf(resolved, i)
		}
	}

	return ""
}

// Like FindKeyboardDevice, but finds all devices which contain keyword 'keyboard'
// Returns an array of file paths which contain keyboard events
func FindAllKeyboardDevices() []string {
	path := "/sys/class/input/event%d/device/name"
	resolved := "/dev/input/event%d"

	valid := make([]string, 0)

	for i := 0; i < 255; i++ {
		buff, err := ioutil.ReadFile(fmt.Sprintf(path, i))

		// prevent from checking non-existant files
		if os.IsNotExist(err) {
			break
		}
		if err != nil {
			continue
		}

		deviceName := strings.ToLower(string(buff))

		if restrictedDevices.hasDevice(deviceName) {
			continue
		} else {
			valid = append(valid, fmt.Sprintf(resolved, i))
		}
	}
	return valid
}

// IsRoot checks if the process is run with root permission
func (k *KeyLogger) IsRoot() bool {
	return syscall.Getuid() == 0 && syscall.Geteuid() == 0
}

// Read from file descriptor
// Blocking call, returns channel
// Make sure to close channel when finish
func (k *KeyLogger) Read() chan mapping.InputEvent {
	event := make(chan mapping.InputEvent)
	go func(event chan mapping.InputEvent) {
		for {
			e, err := k.read()
			if err != nil {
				close(event)
				break
			}

			if e != nil {
				event <- *e
			}
		}
	}(event)
	return event
}

// Write writes to keyboard and sync the event
// This will keep the key pressed or released until you call another write with other direction
// eg, if the key is "A" and direction is press, on UI, you will see "AAAAA..." until you stop with release
// Probably you want to use WriteOnce method
func (k *KeyLogger) Write(direction mapping.KeyEvent, key string) error {
	key = strings.ToUpper(key)
	code := uint16(0)
	for c, k := range mapping.KeyMap {
		if k == key {
			code = c
		}
	}
	if code == 0 {
		return fmt.Errorf("%s key not found in key code map", key)
	}
	err := k.write(mapping.InputEvent{
		Type:  mapping.EvKey,
		Code:  code,
		Value: int32(direction),
	})
	if err != nil {
		return err
	}
	return k.syn()
}

// WriteOnce method simulates single key press
// When you send a key, it will press it, release it and send to sync
func (k *KeyLogger) WriteOnce(key string) error {
	key = strings.ToUpper(key)
	code := uint16(0)
	for c, k := range mapping.KeyMap {
		if k == key {
			code = c
		}
	}
	if code == 0 {
		return fmt.Errorf("%s key not found in key code map", key)
	}

	for _, i := range []int32{int32(mapping.KeyPress), int32(mapping.KeyRelease)} {
		err := k.write(mapping.InputEvent{
			Type:  mapping.EvKey,
			Code:  code,
			Value: i,
		})
		if err != nil {
			return err
		}
	}
	return k.syn()
}

// eventsize is size of structure of InputEvent
var eventsize = int(unsafe.Sizeof(mapping.InputEvent{}))

// read from file description and parse binary into go struct
func (k *KeyLogger) read() (*mapping.InputEvent, error) {
	buffer := make([]byte, eventsize)
	n, err := k.fd.Read(buffer)
	if err != nil {
		return nil, err
	}
	// no input, dont send error
	if n <= 0 {
		return nil, nil
	}
	return k.eventFromBuffer(buffer)
}

// write to keyboard
func (k *KeyLogger) write(ev mapping.InputEvent) error {
	return binary.Write(k.fd, binary.LittleEndian, ev)
}

// syn syncs input events
func (k *KeyLogger) syn() error {
	return binary.Write(k.fd, binary.LittleEndian, mapping.InputEvent{
		Type:  mapping.EvSyn,
		Code:  0,
		Value: 0,
	})
}

// eventFromBuffer parser bytes into mapping.InputEvent struct
func (k *KeyLogger) eventFromBuffer(buffer []byte) (*mapping.InputEvent, error) {
	event := &mapping.InputEvent{}
	err := binary.Read(bytes.NewBuffer(buffer), binary.LittleEndian, event)
	return event, err
}

// Close file descriptor
func (k *KeyLogger) Close() error {
	if k.fd == nil {
		return nil
	}
	return k.fd.Close()
}
