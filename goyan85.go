package main

import (
	"flag"
	"fmt"
	"log"
	"os"
	"syscall"
	"time"

	hexdump "github.com/glycerine/golang-hex-dumper"
)

type Reg struct {
	id  string
	val byte
}

type Inst struct {
	Op   byte
	Arg1 byte
	Arg2 byte
}

const (
	Byte2RegA   byte = 1
	Byte2RegB   byte = 2
	Byte2RegC   byte = 3
	Byte2RegD   byte = 4
	Byte2RegS   byte = 5
	Byte2RegI   byte = 6
	Byte2RegF   byte = 7
	InstImm     byte = 8
	InstAdd     byte = 9
	InstStk     byte = 10
	InstStm     byte = 11
	InstLdm     byte = 12
	InstCmp     byte = 13
	InstJmp     byte = 14
	InstSys     byte = 15
	IndexOp     int  = 0
	IndexArg1   int  = 1
	IndexArg2   int  = 2
	SysOpen     byte = 16
	SysReadCode byte = 17
	SysReadMem  byte = 18
	SysWrite    byte = 19
	SysSleep    byte = 20
	SysExit     byte = 21
	FlagL       byte = 22
	FlagG       byte = 23
	FlagE       byte = 24
	FlagN       byte = 25
	FlagZ       byte = 26
)

type Yan85vm struct {
	RegA     Reg
	RegB     Reg
	RegC     Reg
	RegD     Reg // generic  register
	RegS     Reg // stack    register
	RegI     Reg // program  counter
	RegF     Reg // flag     register
	RegNone  Reg
	Code     [0x300]byte
	Memory   [0x100]byte
	Byte2Reg map[byte]Reg
}

func (vm *Yan85vm) GetReg(b byte) *Reg {
	switch b {
	case Byte2RegA:
		return &vm.RegA
	case Byte2RegB:
		return &vm.RegB
	case Byte2RegC:
		return &vm.RegC
	case Byte2RegD:
		return &vm.RegD
	case Byte2RegS:
		return &vm.RegS
	case Byte2RegI:
		return &vm.RegI
	case Byte2RegF:
		return &vm.RegF
	}
	return &vm.RegNone
}

func FlagDesc(flag byte) string {
	flag_desc := ""
	if flag&FlagL != 0 {
		flag_desc += "L"
	}
	if flag&FlagG != 0 {
		flag_desc += "G"
	}
	if flag&FlagE != 0 {
		flag_desc += "E"
	}
	if flag&FlagN != 0 {
		flag_desc += "N"
	}
	if flag&FlagZ != 0 {
		flag_desc += "Z"
	}
	if flag == 0 {
		flag_desc += "*"
	}
	return flag_desc
}

func (vm *Yan85vm) Execute(inst []Inst) {
	log.Println("[+] GoYan85 VM start! ")
	for {
		pc := int(vm.RegI.val)
		vm.RegI.val += byte(1)
		vm.Interpret(inst[pc])
	}
}

func (vm *Yan85vm) Interpret(inst Inst) {
	switch inst.Op {
	case InstImm:
		vm.Interpret_imm(inst)
	case InstAdd:
		vm.Interpret_add(inst)
	case InstStk:
		vm.Interpret_stk(inst)
	case InstStm:
		vm.Interpret_stm(inst)
	case InstLdm:
		vm.Interpret_ldm(inst)
	case InstCmp:
		vm.Interpret_cmp(inst)
	case InstJmp:
		vm.Interpret_jmp(inst)
	case InstSys:
		vm.Interpret_sys(inst)
	}
}

func (vm *Yan85vm) Interpret_imm(inst Inst) {
	// reg1 = imm
	reg := vm.GetReg(inst.Arg1)
	imm := inst.Arg2
	log.Printf("IMM %s = %#02x\n", reg.id, imm)
	reg.val = imm
}

func (vm *Yan85vm) Interpret_add(inst Inst) {
	// reg1 = reg1 + reg2
	reg1 := vm.GetReg(inst.Arg1)
	reg2 := vm.GetReg(inst.Arg2)
	log.Printf("ADD %s %s\n", reg1.id, reg2.id)
	reg1.val += reg2.val
	log.Printf("Reg %s is %#02x and Reg %s is %#02x now\n", reg1.id, reg1.val, reg2.id, reg2.val)
}

func (vm *Yan85vm) Interpret_stk(inst Inst) {
	// push reg2, pop reg1
	reg1 := vm.GetReg(inst.Arg1)
	reg2 := vm.GetReg(inst.Arg2)
	regS := &vm.RegS
	log.Printf("STK %s %s, pushing %s and poping %s\n", reg1.id, reg2.id, reg1.id, reg2.id)
	if reg2 != &vm.RegNone {
		regS.val += 1
		vm.Memory[regS.val] = reg2.val
	}
	if reg1 != &vm.RegNone {
		reg1.val = vm.Memory[regS.val]
		regS.val -= 1
	}
}

func (vm *Yan85vm) Interpret_stm(inst Inst) {
	// *reg1 = reg2
	reg1 := vm.GetReg(inst.Arg1)
	reg2 := vm.GetReg(inst.Arg2)
	log.Printf("STM *%s = %s\n", reg1.id, reg2.id)
	vm.Memory[reg1.val] = reg2.val
}

func (vm *Yan85vm) Interpret_ldm(inst Inst) {
	// reg1 = *reg2
	reg1 := vm.GetReg(inst.Arg1)
	reg2 := vm.GetReg(inst.Arg2)
	log.Printf("LDM %s = *%s\n", reg1.id, reg2.id)
	reg1.val = vm.Memory[reg2.val]
}

func (vm *Yan85vm) Interpret_cmp(inst Inst) {
	// cmp reg1 & reg2
	reg1 := vm.GetReg(inst.Arg1)
	reg2 := vm.GetReg(inst.Arg2)
	regF := &vm.RegF
	log.Printf("CMP %s:%#02x %s:%#02x\n", reg1.id, reg1.val, reg2.id, reg2.val)
	regF.val = 0
	if reg1.val < reg2.val {
		regF.val |= FlagL
	}
	if reg1.val > reg2.val {
		regF.val |= FlagG
	}
	if reg1.val == reg2.val {
		regF.val |= FlagE
	} else {
		regF.val |= FlagN
	}
	if reg1.val == 0 && reg2.val == 0 {
		regF.val |= FlagZ
	}
	vm.Dump()
	log.Printf("CMP Flag: %s\n", FlagDesc(regF.val))
}

func (vm *Yan85vm) Interpret_jmp(inst Inst) {
	// jmp by flag
	reg := vm.GetReg(inst.Arg2)
	flag := inst.Arg1
	flag_desc := FlagDesc(flag)
	log.Printf("JMP %s %s\n", flag_desc, reg.id)
	if flag == 0 || vm.RegF.val&flag != 0 {
		log.Printf("JMP SUCCESS!")
		vm.RegI.val = reg.val
	} else {
		log.Printf("JMP FAIL!")
	}
}

func (vm *Yan85vm) Interpret_sys(inst Inst) {
	// syscalls, sysflag in reg1, return value in reg2
	reg := vm.GetReg(inst.Arg2)
	sysFlag := inst.Arg1
	log.Printf("SYS %#02x %s\n", sysFlag, reg.id)
	if sysFlag&SysOpen != 0 {
		// sysOpen: open(fn=regA)
		start, end := int(vm.RegA.val), int(vm.RegA.val)
		for i := int(start); i < len(vm.Memory); i++ {
			if vm.Memory[i] == 0 {
				end = i
				break
			}
		}
		filePath := string(vm.Memory[start:end])
		log.Printf("SYS open %s\n", filePath)
		fd, err := syscall.Open(filePath, syscall.O_RDONLY, 0)
		if err != nil {
			log.Fatalf("SYS open error: %v\n", err)
		}
		reg.val = byte(fd)
	}
	if sysFlag&SysReadCode != 0 {
		log.Printf("SYS read_code")
		// sysReadCode: read(fd=regA, buf=Code[3*regB], n=regC)
		fd := int(vm.RegA.val)
		buf := vm.Code[int(vm.RegB.val)*3:]
		n, err := syscall.Read(fd, buf)
		if err != nil {
			log.Fatalf("SYS read_code error: %v\n", err)
		}
		reg.val = byte(n)
	}
	if sysFlag&SysReadMem != 0 {
		fd := int(vm.RegA.val)
		buf := vm.Memory[int(vm.RegB.val):]
		expected := min(0x100-int(vm.RegB.val), int(vm.RegC.val))
		log.Printf("SYS read memory at %#02x for %d bytes", int(vm.RegB.val), expected)
		// sysReadMem: read(fd=regA, buf=Memory[regB], n=regC)
		n, err := syscall.Read(fd, buf)
		if err != nil {
			log.Fatalf("SYS read_memory error: %v\n", err)
		}
		reg.val = byte(n)
	}
	if sysFlag&SysWrite != 0 {
		fd := int(vm.RegA.val)
		n := min(0x100-int(vm.RegB.val), int(vm.RegC.val))
		buf := vm.Memory[int(vm.RegB.val) : int(vm.RegB.val)+n]
		log.Printf("SYS write fd %d from buf at %d for %d bytes", fd, int(vm.RegB.val), n)
		// sysWrite: write(fd=regA, buf=Memory[regB], n=regC)
		res, err := syscall.Write(fd, buf)
		if err != nil {
			log.Fatalf("SYS write error: %v\n", err)
		}
		reg.val = byte(res)
	}
	if sysFlag&SysSleep != 0 {
		log.Printf("SYS sleep for %d seconds\n", vm.RegA.val)
		// sysSleep: sleep(regA)
		time.Sleep(time.Duration(vm.RegA.val) * time.Second)
		reg.val = byte(0)
	}
	if sysFlag&SysExit != 0 {
		log.Printf("SYS exit with return code %d\n", vm.RegA.val)
		// sysExit: exit(regA)
		os.Exit(int(vm.RegA.val))
	}
	vm.Dump()
	if reg != &vm.RegNone {
		log.Printf("SYS return value (in register %s): %#02x\n", reg.id, reg.val)
	}
}

func (vm *Yan85vm) Dump() {
	fmt.Printf("[V] a:%#02x b:%#02x c:%#02x d:%#02x s:%#02x i:%#02x f:%#02x\n", vm.RegA.val, vm.RegB.val, vm.RegC.val, vm.RegD.val, vm.RegS.val, vm.RegI.val, vm.RegF.val)
	hexdump.Dump(vm.Memory[:])
}

func Raw2Code(b []byte) (inst []Inst) {
	N := len(b) - len(b)%3
	for i := 0; i < N; i += 3 {
		inst = append(inst, Inst{b[i+IndexOp], b[i+IndexArg1], b[i+IndexArg2]})
	}
	return
}

func Code2Raw(inst []Inst) (b []byte) {
	for _, v := range inst {
		var snippet [3]byte
		snippet[IndexOp] = v.Op
		snippet[IndexArg1] = v.Arg1
		snippet[IndexArg2] = v.Arg2
		b = append(b, snippet[:]...)
	}
	return
}

func Yan85CatFile(fn string, addr int) (inst []Inst) {
	fn = fn + "\x00"
	inst = append(inst, Inst{InstImm, Byte2RegC, 1})
	inst = append(inst, Inst{InstImm, Byte2RegB, byte(addr)})
	for _, v := range fn {
		inst = append(inst, Inst{InstImm, Byte2RegA, byte(v)})
		inst = append(inst, Inst{InstStm, Byte2RegB, Byte2RegA})
		inst = append(inst, Inst{InstAdd, Byte2RegB, Byte2RegC})
	}
	inst = append(inst, Inst{InstImm, Byte2RegA, byte(addr)})
	inst = append(inst, Inst{InstSys, SysOpen, Byte2RegA})
	inst = append(inst, Inst{InstImm, Byte2RegB, byte(addr)})
	inst = append(inst, Inst{InstImm, Byte2RegC, 255})
	inst = append(inst, Inst{InstSys, SysReadMem, Byte2RegA})
	inst = append(inst, Inst{InstImm, Byte2RegA, 1})
	inst = append(inst, Inst{InstImm, Byte2RegB, byte(addr)})
	inst = append(inst, Inst{InstImm, Byte2RegC, 255})
	inst = append(inst, Inst{InstSys, SysWrite, Byte2RegA})
	inst = append(inst, Inst{InstSys, SysExit, 0})
	return
}

func main() {
	log.SetFlags(0)
	fn := flag.String("file", "", "yan85 code file to execute")
	flag.Parse()
	b, err := os.ReadFile(*fn)
	if err != nil {
		log.Fatalf("Read file error: %v\n", err)
	}
	if len(b) > 0x300 {
		log.Fatal("Too many instructions!")
	}
	code := Raw2Code(b)
	vm := &Yan85vm{
		RegA:    Reg{"a", 0},
		RegB:    Reg{"b", 0},
		RegC:    Reg{"c", 0},
		RegD:    Reg{"d", 0},
		RegS:    Reg{"s", 0},
		RegI:    Reg{"i", 0},
		RegF:    Reg{"f", 0},
		RegNone: Reg{"NONE", 0},
	}
	vm.Execute(code)
}
