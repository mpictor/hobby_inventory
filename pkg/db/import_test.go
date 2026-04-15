package db

import (
	"testing"
)

const (
	ttlCSV = `"prefix","series","family","function","sfx","PN","category","qty","pkg","description","origin","location","notes"
"MC",30,,29,"P","MC3029P","buf/drv/inv",2,,"MTTL III 3-in NAND term line drv","srpls",,
,54,"HC",257,,"54HC257","mux",3,,"quad 2-in mux tristate",,"74 series logic box",
,,,,,,,,,,,,
`
	cmosCSV = `
4xxx and other wide-voltage-range CMOS,,,,,,,,,,,,,,,,,,
mpfx,ord,func,sfx,PN,category,qty,,description,location,interesting,Motorola 1978,comments,,,,,,
CD,40,4001,BCN,CD4001BCN,logic,1,,quad 2-in NOR,digi/ana mux/xtal drawer,,,,,,,,,
FT,40,4001,,FT4001,logic,2,,quad 2-in NOR,asmbly box,,,,,,,,,
,,,,MK50282N,,2,,5 function 8-digit calculator,asmbly box,,mostek,,,,,,,
,,,,UCN5810A,shift reg,5,,BiMOS II serial input latched 10-bit driver 60V ,44xx/45xx cmos box,,,,,,,,,
,,,,UCN5821A,shift reg,5,," 8-bit serial-In, latched HV drivers 50v 500mA OC",44xx/45xx cmos box,,,likely similar: allegro A6821,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,,,,,,,,,,,,,,,
,,,,4017,,5,,decade counter/divider,breadboard,,,,,,,,,HANDWRITTEN
,,,,14049,,16,,hex inv/buffer,44xx/45xx cmos box,,,,,,,,,HANDWRITTEN
,,,,14077,,8,,quad XNOR,44xx/45xx cmos box,,,,,,,,,HANDWRITTEN
,,,,14443,,17,,6-ch 8-10b A/D conv linear subsystem,44xx/45xx cmos box,,7-264,,,,,,,HANDWRITTEN
,,,,14460,,8,,automotive speed ctrl processor,44xx/45xx cmos box,,7-287,flow chart 7-290 (pdf 339),,,,,,HANDWRITTEN
,,,,14490,,17,,hex contact bounce eliminator,44xx/45xx cmos box,,7-326,,,,,,,HANDWRITTEN
,,,,14519,,10,,4-bit AND/OR selector,44xx/45xx cmos box,,7-421,,,,,,,HANDWRITTEN
,,,,14552,,3,,64x4 SRAM,44xx/45xx cmos box,,7-544 (593),,,,,,,HANDWRITTEN
`
	transistorCSV = `
part number,qty,type,N P,V,A,"fT, MHz",hFE,pkg,note,origin,location,alt PN
2N2222,300,BIPOLAR,NPN,30,0.8,250,100,TO-18,"bulk, metal pkg",,discretes box,
2N2222,many,BIPOLAR,NPN,30,0.8,250,100,TO-92,,,transistor drawer,
2N3392,unknown,BIPOLAR,NPN,25,0.5,70,150,TO-92,,,transistor drawer,
2N3704,unknown,BIPOLAR,NPN,30,0.5,100,300,TO-92,,,transistor drawer,
2N9999,0?,BIPOLAR,NPN,>30,0.5,100,300,TO-92,,,transistor drawer,
`
	powerCSV = `
PN,category,qty,V,A,pkg,description,location,
BA546,AUDIO,,,,,see opamp tab,,
MAX1607,CUR LIM,2,,,,USB port current limiter,LDO ziplock,
MAX1930,CUR LIM,2,,,,dual USB port current limiter,LDO ziplock,
`
	dev_kitsCSV = `
dev kit p/n,p/n,qty,conn/skt,desc,mfg,type,origin,location,notes
CP2102N-EK,CP2102N,1,-,Micro-usb → db-9,silabs,Usb-serial eval kit,asmbly member,dev kit box,
CP2102EK,CP2102,1,-,Micro-usb → db-9,silabs,Usb-serial eval kit,asmbly member,dev kit box,sealed`
	diode_tvsCSV = `
"Diodes, TVS, surge",,,,,,,,,,
PN,category,qty,,V,I,"trr, ns",pkg,description,origin,location
MSD6100,array dio,9,,100,0.2,4,TO-92,common cathode,swico,discretes box`
	line_drvCSV = `
mpfx,series,fn,sfx,full pn,qty,n rx,n tx,pkg,proto,drive strength,desc,loc,
MC1,45,407,P,MC145407P,1,3,3,DIP20,,,transciever with charge pump,line driver ziploc,
MAX,,232,,MAX232,7,,,DIP16,,,,line driver ziploc,`
	opampCSV = `
PN,cate gory,class,n/pkg,qty,"Gbp, MHz","slew, V/us",pkg,min Vcc,R-R?,description,origin,location,
LM833,aud,,2,4,15,7,,,,audio,,opamp drawer,
BA546,aud,,1,1,,,,,,6V/430mW single-channel power amplifier,,asmbly box,rohm`
	optoCSV = `
PN,category,qty,,description,location
PS2532,,2,,quad,opamp/555 drawer
LTV846,,5,,quad,opamp/555 drawer`
	otherCSV = `
PN,category,qty,,description,location,origin,notes
NTE7406,TTL,,,HEX INV open collector / high voltage,analog/unique box,estate,
NTE7407,TTL,,,HEX buffer open collector / high voltage,analog/unique box,estate,`
	passiveCSV = `
PN,FUNC,TYP,QTY,value,rating,NOTE,FORM,F FACT,STORAGE,LOC,
,CAP,LYTIC,1800,10u,50V,,TH,,TAPE&BOX,PASSIVE,
,CAP,LYTIC,150,470u,63V,,TH,,BULK,PASSIVE,
BCN164AB103J7,RES,RES ARR,40000,10k,1/16W ea,10Kx4 ARR 5%,SMT,1206,T&R,,`
	tmr_osc_pllCSV = `
PN,category,qty,,description,origin,location
ICM7555,timer,4,,CMOS 555,,ziploc in analog box
MC1455,timer,8,,Equiv 555,,ziploc in analog box`
)

func TestImportCSV(t *testing.T) {

	creates := []string{
		`CREATE TABLE logic(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
datasheet TEXT,
attrs BLOB,
notes TEXT,
location INTEGER NOT NULL,
vrange INTEGER,
prefix TEXT,
series TEXT,
family TEXT,
func TEXT NOT NULL,
sfx TEXT,
category TEXT,
description INTEGER,
FOREIGN KEY (location) REFERENCES locations (id),
FOREIGN KEY (description) REFERENCES logicDescriptions (id)
)`,
		`CREATE TABLE transistors(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
datasheet TEXT,
attrs BLOB,
notes TEXT,
location INTEGER NOT NULL,
pn TEXT NOT NULL,
type TEXT NOT NULL,
doping TEXT NOT NULL,
maxv FLOAT64,
maxa FLOAT64,
maxf FLOAT64,
gain FLOAT64,
altpn TEXT,
FOREIGN KEY (location) REFERENCES locations (id)
)`,
		`CREATE TABLE locations(
id INTEGER PRIMARY KEY,
name TEXT NOT NULL
)`,
		`CREATE TABLE logicdescriptions(
id INTEGER PRIMARY KEY,
desc TEXT NOT NULL
)`,
		`CREATE TABLE dev_kits(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
datasheet TEXT,
attrs BLOB,
notes TEXT,
location INTEGER NOT NULL,
kitpn TEXT NOT NULL,
devicepn TEXT NOT NULL,
connskt TEXT NOT NULL,
desc TEXT NOT NULL,
mfg TEXT NOT NULL,
type TEXT NOT NULL,
FOREIGN KEY (location) REFERENCES locations (id)
)`,
		`CREATE TABLE diode_tvs(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
datasheet TEXT,
attrs BLOB,
notes TEXT,
location INTEGER NOT NULL,
pn TEXT NOT NULL,
category TEXT NOT NULL,
maxv FLOAT64,
maxa FLOAT64,
recoverytime FLOAT64,
description TEXT,
FOREIGN KEY (location) REFERENCES locations (id)
)`,
		`CREATE TABLE line_drv(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
datasheet TEXT,
attrs BLOB,
notes TEXT,
location INTEGER NOT NULL,
mpfx TEXT NOT NULL,
series TEXT NOT NULL,
function TEXT NOT NULL,
sfx TEXT NOT NULL,
n_tx INTEGER,
n_rx INTEGER,
proto TEXT NOT NULL,
drivestrength TEXT NOT NULL,
description TEXT,
FOREIGN KEY (location) REFERENCES locations (id)
)`,
		`CREATE TABLE opamps(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
datasheet TEXT,
attrs BLOB,
notes TEXT,
location INTEGER NOT NULL,
pn TEXT NOT NULL,
category TEXT NOT NULL,
class TEXT NOT NULL,
gainbandwidthproduct FLOAT64,
slew FLOAT64,
minvcc FLOAT64,
railrail TEXT,
description TEXT,
FOREIGN KEY (location) REFERENCES locations (id)
)`,
		`CREATE TABLE opto(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
datasheet TEXT,
attrs BLOB,
notes TEXT,
location INTEGER NOT NULL,
pn TEXT NOT NULL,
category TEXT NOT NULL,
description TEXT,
maxf FLOAT64,
FOREIGN KEY (location) REFERENCES locations (id)
)`,
		`CREATE TABLE others(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
datasheet TEXT,
attrs BLOB,
notes TEXT,
location INTEGER NOT NULL,
pn TEXT NOT NULL,
category TEXT NOT NULL,
description TEXT,
FOREIGN KEY (location) REFERENCES locations (id)
)`,
		`CREATE TABLE passive(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
datasheet TEXT,
attrs BLOB,
notes TEXT,
location INTEGER NOT NULL,
pn TEXT NOT NULL,
function TEXT NOT NULL,
type TEXT NOT NULL,
value TEXT NOT NULL,
rating TEXT NOT NULL,
storage TEXT NOT NULL,
FOREIGN KEY (location) REFERENCES locations (id)
)`,
		`CREATE TABLE power(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
datasheet TEXT,
attrs BLOB,
notes TEXT,
location INTEGER NOT NULL,
pn TEXT NOT NULL,
category TEXT,
maxv FLOAT64,
maxa FLOAT64,
description TEXT,
FOREIGN KEY (location) REFERENCES locations (id)
)`,
		`CREATE TABLE tmr_osc_pll(
id INTEGER PRIMARY KEY,
qty INTEGER NOT NULL,
npkg INTEGER NOT NULL,
package TEXT,
mounting INTEGER,
origin TEXT,
datasheet TEXT,
attrs BLOB,
notes TEXT,
location INTEGER NOT NULL,
pn TEXT NOT NULL,
category TEXT NOT NULL,
description TEXT,
maxf FLOAT64,
FOREIGN KEY (location) REFERENCES locations (id)
)`,
		// ``,
		// ``,
	}

	testdata := map[string]struct {
		tbl        mainDBtbl
		csv        string
		statements []string
	}{
		"ttl": {
			tbl: &Logic{},
			csv: ttlCSV,
			statements: []string{
				"SELECT id FROM logicDescriptions WHERE desc='MTTL III 3-in NAND term line drv'",
				"INSERT INTO logicDescriptions (desc) VALUES('MTTL III 3-in NAND term line drv')",
				"SELECT id FROM logicDescriptions WHERE desc='quad 2-in mux tristate'",
				"INSERT INTO logicDescriptions (desc) VALUES('quad 2-in mux tristate')",

				"SELECT id FROM locations WHERE name='74 series logic box'",
				"INSERT INTO locations (name) VALUES('74 series logic box')",

				"INSERT INTO logic (qty,npkg,mounting,location,origin,vrange,prefix,series,func,sfx,category,description) VALUES(?,?,?,?,?,?,?,?,?,?,?,?); {2,1,0,0,srpls,1,MC,30,29,P,buf/drv/inv,1}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,series,family,func,category,description) VALUES(?,?,?,?,?,?,?,?,?,?); {3,4,0,1,1,54,HC,257,mux,2}",
			},
		},
		"cmos": {
			tbl: &Logic{},
			csv: cmosCSV,
			statements: []string{
				"SELECT id FROM logicDescriptions WHERE desc='quad 2-in NOR'",
				"INSERT INTO logicDescriptions (desc) VALUES('quad 2-in NOR')",
				"SELECT id FROM locations WHERE name='digi/ana mux/xtal drawer'",
				"INSERT INTO locations (name) VALUES('digi/ana mux/xtal drawer')",
				"SELECT id FROM logicDescriptions WHERE desc='quad 2-in NOR'",
				"SELECT id FROM locations WHERE name='asmbly box'",
				"INSERT INTO locations (name) VALUES('asmbly box')",
				"SELECT id FROM logicDescriptions WHERE desc='5 function 8-digit calculator'",
				"INSERT INTO logicDescriptions (desc) VALUES('5 function 8-digit calculator')",
				"SELECT id FROM locations WHERE name='asmbly box'",
				"SELECT id FROM logicDescriptions WHERE desc='BiMOS II serial input latched 10-bit driver 60V '",
				"INSERT INTO logicDescriptions (desc) VALUES('BiMOS II serial input latched 10-bit driver 60V ')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"INSERT INTO locations (name) VALUES('44xx/45xx cmos box')",
				"SELECT id FROM logicDescriptions WHERE desc=' 8-bit serial-In, latched HV drivers 50v 500mA OC'",
				"INSERT INTO logicDescriptions (desc) VALUES(' 8-bit serial-In, latched HV drivers 50v 500mA OC')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='decade counter/divider'",
				"INSERT INTO logicDescriptions (desc) VALUES('decade counter/divider')",
				"SELECT id FROM locations WHERE name='breadboard'",
				"INSERT INTO locations (name) VALUES('breadboard')",
				"SELECT id FROM logicDescriptions WHERE desc='hex inv/buffer'",
				"INSERT INTO logicDescriptions (desc) VALUES('hex inv/buffer')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='quad XNOR'",
				"INSERT INTO logicDescriptions (desc) VALUES('quad XNOR')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='6-ch 8-10b A/D conv linear subsystem'",
				"INSERT INTO logicDescriptions (desc) VALUES('6-ch 8-10b A/D conv linear subsystem')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='automotive speed ctrl processor'",
				"INSERT INTO logicDescriptions (desc) VALUES('automotive speed ctrl processor')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='hex contact bounce eliminator'",
				"INSERT INTO logicDescriptions (desc) VALUES('hex contact bounce eliminator')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='4-bit AND/OR selector'",
				"INSERT INTO logicDescriptions (desc) VALUES('4-bit AND/OR selector')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"SELECT id FROM logicDescriptions WHERE desc='64x4 SRAM'",
				"INSERT INTO logicDescriptions (desc) VALUES('64x4 SRAM')",
				"SELECT id FROM locations WHERE name='44xx/45xx cmos box'",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,prefix,series,func,sfx,category,description) VALUES(?,?,?,?,?,?,?,?,?,?,?); {1,4,0,1,2,CD,40,4001,BCN,logic,1}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,prefix,series,func,category,description) VALUES(?,?,?,?,?,?,?,?,?,?); {2,4,0,2,2,FT,40,4001,logic,1}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,vrange,func,description) VALUES(?,?,?,?,?,?,?,?); {2,1,0,2,moto78:mostek,2,MK50282N,2}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,func,category,description) VALUES(?,?,?,?,?,?,?,?); {5,1,0,3,2,UCN5810A,shift reg,3}",
				"INSERT INTO logic (qty,npkg,mounting,location,notes,vrange,func,category,description) VALUES(?,?,?,?,?,?,?,?,?); {5,1,0,3,likely similar: allegro A6821,2,UCN5821A,shift reg,4}",
				// "INSERT INTO logic (qty,npkg,mounting,location,vrange,func) VALUES(?,?,?,?,?,?); {0,0,0,0,2,}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,func,description) VALUES(?,?,?,?,?,?,?); {5,1,0,4,2,4017,5}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,func,description) VALUES(?,?,?,?,?,?,?); {16,6,0,3,2,14049,6}",
				"INSERT INTO logic (qty,npkg,mounting,location,vrange,func,description) VALUES(?,?,?,?,?,?,?); {8,4,0,3,2,14077,7}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,vrange,func,description) VALUES(?,?,?,?,?,?,?,?); {17,1,0,3,moto78:7-264,2,14443,8}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,notes,vrange,func,description) VALUES(?,?,?,?,?,?,?,?,?); {8,1,0,3,moto78:7-287,flow chart 7-290 (pdf 339);HANDWRITTEN,2,14460,9}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,vrange,func,description) VALUES(?,?,?,?,?,?,?,?); {17,6,0,3,moto78:7-326,2,14490,10}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,vrange,func,description) VALUES(?,?,?,?,?,?,?,?); {10,1,0,3,moto78:7-421,2,14519,11}",
				"INSERT INTO logic (qty,npkg,mounting,location,datasheet,vrange,func,description) VALUES(?,?,?,?,?,?,?,?); {3,1,0,3,moto78:7-544 (593),2,14552,12}",
			},
		},
		"transistors": {
			tbl: &Transistors{},
			csv: transistorCSV,
			statements: []string{
				"SELECT id FROM locations WHERE name='discretes box'",
				"INSERT INTO locations (name) VALUES('discretes box')",
				"SELECT id FROM locations WHERE name='transistor drawer'",
				"INSERT INTO locations (name) VALUES('transistor drawer')",
				"SELECT id FROM locations WHERE name='transistor drawer'",
				"SELECT id FROM locations WHERE name='transistor drawer'",
				"SELECT id FROM locations WHERE name='transistor drawer'",
				"INSERT INTO transistors (qty,npkg,mounting,location,package,notes,pn,type,doping,maxv,maxa,maxf,gain) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?); {300,1,0,1,TO-18,bulk, metal pkg,2N2222,BIPOLAR,NPN,30,0.8,2.5e+08,100}",
				"INSERT INTO transistors (qty,npkg,mounting,location,package,pn,type,doping,maxv,maxa,maxf,gain) VALUES(?,?,?,?,?,?,?,?,?,?,?,?); {-99,1,0,2,TO-92,2N2222,BIPOLAR,NPN,30,0.8,2.5e+08,100}",
				"INSERT INTO transistors (qty,npkg,mounting,location,package,pn,type,doping,maxv,maxa,maxf,gain) VALUES(?,?,?,?,?,?,?,?,?,?,?,?); {-999,1,0,2,TO-92,2N3392,BIPOLAR,NPN,25,0.5,7e+07,150}",
				"INSERT INTO transistors (qty,npkg,mounting,location,package,pn,type,doping,maxv,maxa,maxf,gain) VALUES(?,?,?,?,?,?,?,?,?,?,?,?); {-999,1,0,2,TO-92,2N3704,BIPOLAR,NPN,30,0.5,1e+08,300}",
				"INSERT INTO transistors (qty,npkg,mounting,location,package,pn,type,doping,maxv,maxa,maxf,gain) VALUES(?,?,?,?,?,?,?,?,?,?,?,?); {-999,1,0,2,TO-92,2N9999,BIPOLAR,NPN,30,0.5,1e+08,300}",
			},
		},
		"diode_tvs": {
			tbl: &Diode_TVSDevs{},
			csv: diode_tvsCSV,
			statements: []string{
				"SELECT id FROM locations WHERE name='discretes box'",
				"INSERT INTO locations (name) VALUES('discretes box')",
				"INSERT INTO diode_tvs (qty,npkg,mounting,location,package,origin,pn,category,maxv,maxa,recoveryTime,description) VALUES(?,?,?,?,?,?,?,?,?,?,?,?); {9,0,0,1,TO-92,swico,MSD6100,array dio,100,0.2,4,common cathode}",
			},
		},
		"line_drv": {
			tbl: &Line_Drv{},
			csv: line_drvCSV,
			statements: []string{
				"SELECT id FROM locations WHERE name='line driver ziploc'",
				"INSERT INTO locations (name) VALUES('line driver ziploc')",
				"SELECT id FROM locations WHERE name='line driver ziploc'",
				"INSERT INTO line_drv (qty,npkg,mounting,location,package,notes,mpfx,series,function,sfx,n_tx,n_rx,proto,driveStrength,description) VALUES(?,?,?,?,?,?,?,?,?,?,?,?,?,?,?); {1,0,0,1,DIP20,,MC1,45,407,P,3,3,,,transciever with charge pump}",
				"INSERT INTO line_drv (qty,npkg,mounting,location,package,notes,mpfx,series,function,sfx,proto,driveStrength) VALUES(?,?,?,?,?,?,?,?,?,?,?,?); {7,0,0,1,DIP16,,MAX,,232,,,}",
			},
		},
		"opamp": {
			tbl: &Opamps{},
			csv: opampCSV,
			statements: []string{
				"SELECT id FROM locations WHERE name='opamp drawer'",
				"INSERT INTO locations (name) VALUES('opamp drawer')",
				"SELECT id FROM locations WHERE name='asmbly box'",
				"INSERT INTO locations (name) VALUES('asmbly box')",
				"INSERT INTO opamps (qty,npkg,mounting,location,notes,pn,category,class,gainBandwidthProduct,slew,description) VALUES(?,?,?,?,?,?,?,?,?,?,?); {4,2,0,1,,LM833,aud,,15,7,audio}",
				"INSERT INTO opamps (qty,npkg,mounting,location,notes,pn,category,class,description) VALUES(?,?,?,?,?,?,?,?,?); {1,1,0,2,rohm,BA546,aud,,6V/430mW single-channel power amplifier}",
			},
		},
		"opto": {
			tbl: &OptoDevs{},
			csv: optoCSV,
			statements: []string{
				"SELECT id FROM locations WHERE name='opamp/555 drawer'",
				"INSERT INTO locations (name) VALUES('opamp/555 drawer')",
				"SELECT id FROM locations WHERE name='opamp/555 drawer'",
				"INSERT INTO opto (qty,npkg,mounting,location,pn,category,description) VALUES(?,?,?,?,?,?,?); {2,0,0,1,PS2532,,quad}",
				"INSERT INTO opto (qty,npkg,mounting,location,pn,category,description) VALUES(?,?,?,?,?,?,?); {5,0,0,1,LTV846,,quad}",
			},
		},
		"other": {
			tbl: &Others{},
			csv: otherCSV,
			statements: []string{
				"SELECT id FROM locations WHERE name='analog/unique box'",
				"INSERT INTO locations (name) VALUES('analog/unique box')",
				"SELECT id FROM locations WHERE name='analog/unique box'",
				"INSERT INTO others (qty,npkg,mounting,location,origin,notes,pn,category,description) VALUES(?,?,?,?,?,?,?,?,?); {-999,0,0,1,estate,,NTE7406,TTL,HEX INV open collector / high voltage}",
				"INSERT INTO others (qty,npkg,mounting,location,origin,notes,pn,category,description) VALUES(?,?,?,?,?,?,?,?,?); {-999,0,0,1,estate,,NTE7407,TTL,HEX buffer open collector / high voltage}"},
		},
		"passive": {
			tbl: &PassiveDevs{},
			csv: passiveCSV,
			statements: []string{
				"SELECT id FROM locations WHERE name='PASSIVE'",
				"INSERT INTO locations (name) VALUES('PASSIVE')",
				"SELECT id FROM locations WHERE name='PASSIVE'",
				"SELECT id FROM locations WHERE name=''",
				"INSERT INTO locations (name) VALUES('')",
				"INSERT INTO passive (qty,npkg,mounting,location,notes,pn,function,type,value,rating,storage) VALUES(?,?,?,?,?,?,?,?,?,?,?); {1800,0,3,1,,,CAP,LYTIC,10u,50V,TAPE&BOX}",
				"INSERT INTO passive (qty,npkg,mounting,location,notes,pn,function,type,value,rating,storage) VALUES(?,?,?,?,?,?,?,?,?,?,?); {150,0,3,1,,,CAP,LYTIC,470u,63V,BULK}",
				"INSERT INTO passive (qty,npkg,mounting,location,package,notes,pn,function,type,value,rating,storage) VALUES(?,?,?,?,?,?,?,?,?,?,?,?); {40000,0,2,2,1206,10Kx4 ARR 5%,BCN164AB103J7,RES,RES ARR,10k,1/16W ea,T&R}",
			},
		},
		"power": {
			tbl: &PowerDevs{},
			csv: powerCSV,
			statements: []string{
				"SELECT id FROM locations WHERE name=''",
				"INSERT INTO locations (name) VALUES('')",
				"SELECT id FROM locations WHERE name='LDO ziplock'",
				"INSERT INTO locations (name) VALUES('LDO ziplock')",
				"SELECT id FROM locations WHERE name='LDO ziplock'",
				"INSERT INTO power (qty,npkg,mounting,location,pn,category,description) VALUES(?,?,?,?,?,?,?); {-999,0,0,1,BA546,AUDIO,see opamp tab}",
				"INSERT INTO power (qty,npkg,mounting,location,pn,category,description) VALUES(?,?,?,?,?,?,?); {2,0,0,2,MAX1607,CUR LIM,USB port current limiter}",
				"INSERT INTO power (qty,npkg,mounting,location,pn,category,description) VALUES(?,?,?,?,?,?,?); {2,0,0,2,MAX1930,CUR LIM,dual USB port current limiter}",
			},
		},
		"tmr_osc_pll": {
			tbl: &Tmr_Osc_PllDevs{},
			csv: tmr_osc_pllCSV,
			statements: []string{
				"SELECT id FROM locations WHERE name='ziploc in analog box'",
				"INSERT INTO locations (name) VALUES('ziploc in analog box')",
				"SELECT id FROM locations WHERE name='ziploc in analog box'",
				"INSERT INTO tmr_osc_pll (qty,npkg,mounting,location,pn,category,description) VALUES(?,?,?,?,?,?,?); {4,0,0,1,ICM7555,timer,CMOS 555}",
				"INSERT INTO tmr_osc_pll (qty,npkg,mounting,location,pn,category,description) VALUES(?,?,?,?,?,?,?); {8,0,0,1,MC1455,timer,Equiv 555}",
			},
		},
		"dev_kits": {
			tbl: &Dev_kits{},
			csv: dev_kitsCSV,
			statements: []string{
				"SELECT id FROM locations WHERE name='dev kit box'",
				"INSERT INTO locations (name) VALUES('dev kit box')",
				"SELECT id FROM locations WHERE name='dev kit box'",
				"INSERT INTO dev_kits (qty,npkg,mounting,location,origin,notes,kitpn,devicepn,connSkt,desc,mfg,type) VALUES(?,?,?,?,?,?,?,?,?,?,?,?); {1,0,0,1,asmbly member,,CP2102N-EK,CP2102N,-,Micro-usb → db-9,silabs,Usb-serial eval kit}",
				"INSERT INTO dev_kits (qty,npkg,mounting,location,origin,notes,kitpn,devicepn,connSkt,desc,mfg,type) VALUES(?,?,?,?,?,?,?,?,?,?,?,?); {1,0,0,1,asmbly member,sealed,CP2102EK,CP2102,-,Micro-usb → db-9,silabs,Usb-serial eval kit}",
			},
		},
		// "util":{}, // nothing to import? TODO
	}
	for name, td := range testdata {
		t.Run(name, func(t *testing.T) {
			if td.tbl == nil {
				t.Fatal("table undefined")
			}
			tbl := td.tbl
			db, h, _ := testDB(t)
			if err := tbl.ImportCSV(db, name, []byte(td.csv)); err != nil {
				t.Fatal(err)
			}
			if err := tbl.Insert(db); err != nil {
				t.Fatal(err)
			}
			stmts := creates
			stmts = append(stmts, td.statements...)
			h.checkStatements(t, stmts)
		})
	}
}
