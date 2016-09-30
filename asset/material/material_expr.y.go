//line material_expr.y:2

//go:generate go tool yacc -o material_expr.y.go -p expr material_expr.y
package material

import __yyfmt__ "fmt"

//line material_expr.y:3
import (
	"bytes"
	"fmt"
	"strconv"
	"unicode/utf8"
)

//line material_expr.y:14
type exprSymType struct {
	yys  int
	cVal rune
	fVal float32
	sVal string
	node ExprNode
}

const tokLPAREN = 57346
const tokRPAREN = 57347
const tokLCURLY = 57348
const tokRCURLY = 57349
const tokCOMMA = 57350
const tokCOLON = 57351
const tokFLOAT = 57352
const tokMATERIAL_NAME = 57353
const tokTEXTURE = 57354
const tokREFLECTANCE = 57355
const tokSPECULARITY = 57356
const tokTRANSMITTANCE = 57357
const tokRADIANCE = 57358
const tokINT_IOR = 57359
const tokEXT_IOR = 57360
const tokSCALE = 57361
const tokROUGHNESS = 57362
const tokDIFFUSE = 57363
const tokCONDUCTOR = 57364
const tokROUGH_CONDUCTOR = 57365
const tokDIELECTRIC = 57366
const tokROUGH_DIELECTRIC = 57367
const tokEMISSIVE = 57368
const tokMIX = 57369
const tokMIX_MAP = 57370
const tokBUMP_MAP = 57371
const tokNORMAL_MAP = 57372
const tokDISPERSE = 57373

var exprToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"tokLPAREN",
	"tokRPAREN",
	"tokLCURLY",
	"tokRCURLY",
	"tokCOMMA",
	"tokCOLON",
	"tokFLOAT",
	"tokMATERIAL_NAME",
	"tokTEXTURE",
	"tokREFLECTANCE",
	"tokSPECULARITY",
	"tokTRANSMITTANCE",
	"tokRADIANCE",
	"tokINT_IOR",
	"tokEXT_IOR",
	"tokSCALE",
	"tokROUGHNESS",
	"tokDIFFUSE",
	"tokCONDUCTOR",
	"tokROUGH_CONDUCTOR",
	"tokDIELECTRIC",
	"tokROUGH_DIELECTRIC",
	"tokEMISSIVE",
	"tokMIX",
	"tokMIX_MAP",
	"tokBUMP_MAP",
	"tokNORMAL_MAP",
	"tokDISPERSE",
}
var exprStatenames = [...]string{}

const exprEofCode = 1
const exprErrCode = 2
const exprInitialStackSize = 16

//line material_expr.y:177

// The parser expects the lexer to return 0 on kEOF.
const tokEOF = 0

// The parser uses the type <prefix>Lex as a lexer.  It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type matExprLexer struct {
	line     []byte
	peek     rune
	tokenBuf bytes.Buffer

	parsedExpression ExprNode

	lastError error
}

// The parser calls this method to get each new token.
func (x *matExprLexer) Lex(yylval *exprSymType) int {
	for {
		c := x.next()
		switch c {
		case tokEOF:
			return tokEOF
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9', '.':
			x.tokenBuf.Reset()
			return x.lexFloat32(c, yylval)
		case '"':
			x.tokenBuf.Reset()
			return x.lexLiteral(c, yylval)
		case '(':
			return tokLPAREN
		case ')':
			return tokRPAREN
		case ',':
			return tokCOMMA
		case '{':
			return tokLCURLY
		case '}':
			return tokRCURLY
		case ':':
			return tokCOLON
		case ' ', '\t', '\n', '\r':
		default:
			x.tokenBuf.Reset()
			return x.lexIdentifier(c, yylval)
		}
	}
}

// Lex a float.
func (x *matExprLexer) lexFloat32(c rune, yylval *exprSymType) int {
	x.captureRune(c)
	for {
		c = x.next()
		if (c >= '0' && c <= '9') || (c == '.' || c == 'e' || c == 'E' || c == '+' || c == '-') {
			x.captureRune(c)
			continue
		}
		break
	}

	if c != tokEOF {
		x.peek = c
	}

	val, err := strconv.ParseFloat(x.tokenBuf.String(), 32)
	if err != nil {
		x.Error(fmt.Sprintf("invalid float value %q: %v", x.tokenBuf.String(), err))
		return tokEOF
	}

	yylval.fVal = float32(val)
	return tokFLOAT
}

// Lex literal.
func (x *matExprLexer) lexLiteral(c rune, yylval *exprSymType) int {
	for {
		c = x.next()
		if c == tokEOF || c == '"' {
			break
		}

		x.captureRune(c)
	}

	if c == tokEOF {
		x.Error("unterminated string litera")
		return tokEOF
	}

	yylval.sVal = x.tokenBuf.String()
	if supportedImageRegex.MatchString(yylval.sVal) {
		return tokTEXTURE
	}

	return tokMATERIAL_NAME
}

// Lex identifier.
func (x *matExprLexer) lexIdentifier(c rune, yylval *exprSymType) int {
	x.captureRune(c)
	for {
		c = x.next()
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || c == '_' {
			x.captureRune(c)
			continue
		}
		break
	}

	if c != tokEOF {
		x.peek = c
	}

	yylval.sVal = x.tokenBuf.String()

	switch yylval.sVal {
	// BXDFS
	case "diffuse":
		return tokDIFFUSE
	case "conductor":
		return tokCONDUCTOR
	case "roughConductor":
		return tokROUGH_CONDUCTOR
	case "dielectric":
		return tokDIELECTRIC
	case "roughDielectric":
		return tokROUGH_DIELECTRIC
	case "emissive":
		return tokEMISSIVE
	// Operators
	case "mix":
		return tokMIX
	case "mixMap":
		return tokMIX_MAP
	case "bumpMap":
		return tokBUMP_MAP
	case "normalMap":
		return tokNORMAL_MAP
	case "disperse":
		return tokDISPERSE
	// Parameters
	case ParamReflectance:
		return tokREFLECTANCE
	case ParamSpecularity:
		return tokSPECULARITY
	case ParamTransmittance:
		return tokTRANSMITTANCE
	case ParamRadiance:
		return tokRADIANCE
	case ParamIntIOR:
		return tokINT_IOR
	case ParamExtIOR:
		return tokEXT_IOR
	case ParamScale:
		return tokSCALE
	case ParamRoughness:
		return tokROUGHNESS
	default:
		x.Error(fmt.Sprintf("invalid expression %q", yylval.sVal))
		return tokEOF
	}
}

// Append a rune to the token buffer.
func (x *matExprLexer) captureRune(c rune) {
	x.tokenBuf.WriteRune(c)
}

// Return the next rune for the lexer.
func (x *matExprLexer) next() rune {
	if x.peek != tokEOF {
		r := x.peek
		x.peek = tokEOF
		return r
	}
	if len(x.line) == 0 {
		return tokEOF
	}
	c, size := utf8.DecodeRune(x.line)
	x.line = x.line[size:]
	if c == utf8.RuneError && size == 1 {
		x.Error("encountered invalid utf8 rune")
		return x.next()
	}
	return c
}

// The parser calls this method on a parse error.
func (x *matExprLexer) Error(s string) {
	// tokKeep the first error we encountered
	if x.lastError == nil {
		x.lastError = fmt.Errorf(s)
	}
}

// Parser interface.
func ParseExpression(input string) (ExprNode, error) {
	matLexer := &matExprLexer{line: []byte(input)}
	exprNewParser().Parse(matLexer)
	if matLexer.lastError != nil {
		return nil, matLexer.lastError
	}

	return matLexer.parsedExpression, nil
}

//line yacctab:1
var exprExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const exprNprod = 37
const exprPrivate = 57344

var exprTokenNames []string
var exprStates []string

const exprLast = 111

var exprAct = [...]int{

	58, 33, 64, 57, 24, 25, 26, 27, 28, 29,
	30, 31, 32, 93, 36, 76, 70, 85, 71, 75,
	37, 38, 39, 40, 10, 11, 12, 13, 14, 15,
	5, 6, 7, 8, 9, 10, 11, 12, 13, 14,
	15, 5, 6, 7, 8, 9, 60, 56, 61, 62,
	63, 67, 59, 72, 73, 74, 65, 66, 94, 92,
	87, 84, 77, 68, 96, 82, 50, 49, 48, 47,
	46, 45, 44, 43, 91, 90, 88, 83, 79, 78,
	55, 54, 53, 86, 52, 51, 42, 97, 60, 99,
	95, 89, 81, 80, 41, 21, 20, 98, 19, 18,
	17, 16, 34, 2, 35, 3, 4, 23, 22, 69,
	1,
}
var exprPact = [...]int{

	14, -1000, -1000, -1000, 97, 96, 95, 94, 92, 91,
	-1000, -1000, -1000, -1000, -1000, -1000, -8, 3, 3, 3,
	3, 3, 89, 78, -1000, 64, 63, 62, 61, 60,
	59, 58, 57, 77, -1000, -1000, -1000, 76, 74, 73,
	72, -1000, -8, 40, 40, 40, 40, 46, 46, 53,
	6, 3, 3, 43, 7, -2, -1000, -1000, -1000, -1000,
	52, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000,
	-1000, -1000, 71, 70, 88, 87, 56, 69, 51, 5,
	-1000, -1000, 82, 50, 68, 86, 67, 66, 49, -1000,
	-5, 48, 85, 55, 80, -1000, 82, -1000, 84, -1000,
}
var exprPgo = [...]int{

	0, 110, 0, 4, 3, 2, 109, 104, 108, 107,
	102, 1, 106,
}
var exprR1 = [...]int{

	0, 1, 1, 10, 12, 12, 12, 12, 12, 12,
	8, 8, 9, 9, 3, 3, 3, 3, 3, 3,
	3, 3, 4, 4, 2, 5, 5, 6, 6, 7,
	7, 7, 7, 7, 11, 11, 11,
}
var exprR2 = [...]int{

	0, 1, 1, 4, 1, 1, 1, 1, 1, 1,
	0, 1, 1, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 1, 1, 7, 1, 1, 1, 1, 10,
	8, 6, 6, 12, 1, 1, 1,
}
var exprChk = [...]int{

	-1000, -1, -10, -7, -12, 27, 28, 29, 30, 31,
	21, 22, 23, 24, 25, 26, 4, 4, 4, 4,
	4, 4, -8, -9, -3, 13, 14, 15, 16, 17,
	18, 19, 20, -11, -10, -7, 11, -11, -11, -11,
	-11, 5, 8, 9, 9, 9, 9, 9, 9, 9,
	9, 8, 8, 8, 8, 8, -3, -4, -2, 12,
	6, -4, -4, -4, -5, 10, 11, -5, 10, -6,
	10, 12, -11, -11, 12, 12, 17, 10, 8, 8,
	5, 5, 9, 8, 10, 12, -2, 10, 8, 5,
	8, 8, 10, 18, 10, 5, 9, 7, -2, 5,
}
var exprDef = [...]int{

	0, -2, 1, 2, 0, 0, 0, 0, 0, 0,
	4, 5, 6, 7, 8, 9, 10, 0, 0, 0,
	0, 0, 0, 11, 12, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 34, 35, 36, 0, 0, 0,
	0, 3, 0, 0, 0, 0, 0, 0, 0, 0,
	0, 0, 0, 0, 0, 0, 13, 14, 22, 23,
	0, 15, 16, 17, 18, 25, 26, 19, 20, 21,
	27, 28, 0, 0, 0, 0, 0, 0, 0, 0,
	31, 32, 0, 0, 0, 0, 0, 0, 0, 30,
	0, 0, 0, 0, 0, 29, 0, 24, 0, 33,
}
var exprTok1 = [...]int{

	1,
}
var exprTok2 = [...]int{

	2, 3, 4, 5, 6, 7, 8, 9, 10, 11,
	12, 13, 14, 15, 16, 17, 18, 19, 20, 21,
	22, 23, 24, 25, 26, 27, 28, 29, 30, 31,
}
var exprTok3 = [...]int{
	0,
}

var exprErrorMessages = [...]struct {
	state int
	token int
	msg   string
}{}

//line yaccpar:1

/*	parser for yacc output	*/

var (
	exprDebug        = 0
	exprErrorVerbose = false
)

type exprLexer interface {
	Lex(lval *exprSymType) int
	Error(s string)
}

type exprParser interface {
	Parse(exprLexer) int
	Lookahead() int
}

type exprParserImpl struct {
	lval  exprSymType
	stack [exprInitialStackSize]exprSymType
	char  int
}

func (p *exprParserImpl) Lookahead() int {
	return p.char
}

func exprNewParser() exprParser {
	return &exprParserImpl{}
}

const exprFlag = -1000

func exprTokname(c int) string {
	if c >= 1 && c-1 < len(exprToknames) {
		if exprToknames[c-1] != "" {
			return exprToknames[c-1]
		}
	}
	return __yyfmt__.Sprintf("tok-%v", c)
}

func exprStatname(s int) string {
	if s >= 0 && s < len(exprStatenames) {
		if exprStatenames[s] != "" {
			return exprStatenames[s]
		}
	}
	return __yyfmt__.Sprintf("state-%v", s)
}

func exprErrorMessage(state, lookAhead int) string {
	const TOKSTART = 4

	if !exprErrorVerbose {
		return "syntax error"
	}

	for _, e := range exprErrorMessages {
		if e.state == state && e.token == lookAhead {
			return "syntax error: " + e.msg
		}
	}

	res := "syntax error: unexpected " + exprTokname(lookAhead)

	// To match Bison, suggest at most four expected tokens.
	expected := make([]int, 0, 4)

	// Look for shiftable tokens.
	base := exprPact[state]
	for tok := TOKSTART; tok-1 < len(exprToknames); tok++ {
		if n := base + tok; n >= 0 && n < exprLast && exprChk[exprAct[n]] == tok {
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}
	}

	if exprDef[state] == -2 {
		i := 0
		for exprExca[i] != -1 || exprExca[i+1] != state {
			i += 2
		}

		// Look for tokens that we accept or reduce.
		for i += 2; exprExca[i] >= 0; i += 2 {
			tok := exprExca[i]
			if tok < TOKSTART || exprExca[i+1] == 0 {
				continue
			}
			if len(expected) == cap(expected) {
				return res
			}
			expected = append(expected, tok)
		}

		// If the default action is to accept or reduce, give up.
		if exprExca[i+1] != 0 {
			return res
		}
	}

	for i, tok := range expected {
		if i == 0 {
			res += ", expecting "
		} else {
			res += " or "
		}
		res += exprTokname(tok)
	}
	return res
}

func exprlex1(lex exprLexer, lval *exprSymType) (char, token int) {
	token = 0
	char = lex.Lex(lval)
	if char <= 0 {
		token = exprTok1[0]
		goto out
	}
	if char < len(exprTok1) {
		token = exprTok1[char]
		goto out
	}
	if char >= exprPrivate {
		if char < exprPrivate+len(exprTok2) {
			token = exprTok2[char-exprPrivate]
			goto out
		}
	}
	for i := 0; i < len(exprTok3); i += 2 {
		token = exprTok3[i+0]
		if token == char {
			token = exprTok3[i+1]
			goto out
		}
	}

out:
	if token == 0 {
		token = exprTok2[1] /* unknown char */
	}
	if exprDebug >= 3 {
		__yyfmt__.Printf("lex %s(%d)\n", exprTokname(token), uint(char))
	}
	return char, token
}

func exprParse(exprlex exprLexer) int {
	return exprNewParser().Parse(exprlex)
}

func (exprrcvr *exprParserImpl) Parse(exprlex exprLexer) int {
	var exprn int
	var exprVAL exprSymType
	var exprDollar []exprSymType
	_ = exprDollar // silence set and not used
	exprS := exprrcvr.stack[:]

	Nerrs := 0   /* number of errors */
	Errflag := 0 /* error recovery flag */
	exprstate := 0
	exprrcvr.char = -1
	exprtoken := -1 // exprrcvr.char translated into internal numbering
	defer func() {
		// Make sure we report no lookahead when not parsing.
		exprstate = -1
		exprrcvr.char = -1
		exprtoken = -1
	}()
	exprp := -1
	goto exprstack

ret0:
	return 0

ret1:
	return 1

exprstack:
	/* put a state and value onto the stack */
	if exprDebug >= 4 {
		__yyfmt__.Printf("char %v in %v\n", exprTokname(exprtoken), exprStatname(exprstate))
	}

	exprp++
	if exprp >= len(exprS) {
		nyys := make([]exprSymType, len(exprS)*2)
		copy(nyys, exprS)
		exprS = nyys
	}
	exprS[exprp] = exprVAL
	exprS[exprp].yys = exprstate

exprnewstate:
	exprn = exprPact[exprstate]
	if exprn <= exprFlag {
		goto exprdefault /* simple state */
	}
	if exprrcvr.char < 0 {
		exprrcvr.char, exprtoken = exprlex1(exprlex, &exprrcvr.lval)
	}
	exprn += exprtoken
	if exprn < 0 || exprn >= exprLast {
		goto exprdefault
	}
	exprn = exprAct[exprn]
	if exprChk[exprn] == exprtoken { /* valid shift */
		exprrcvr.char = -1
		exprtoken = -1
		exprVAL = exprrcvr.lval
		exprstate = exprn
		if Errflag > 0 {
			Errflag--
		}
		goto exprstack
	}

exprdefault:
	/* default state action */
	exprn = exprDef[exprstate]
	if exprn == -2 {
		if exprrcvr.char < 0 {
			exprrcvr.char, exprtoken = exprlex1(exprlex, &exprrcvr.lval)
		}

		/* look through exception table */
		xi := 0
		for {
			if exprExca[xi+0] == -1 && exprExca[xi+1] == exprstate {
				break
			}
			xi += 2
		}
		for xi += 2; ; xi += 2 {
			exprn = exprExca[xi+0]
			if exprn < 0 || exprn == exprtoken {
				break
			}
		}
		exprn = exprExca[xi+1]
		if exprn < 0 {
			goto ret0
		}
	}
	if exprn == 0 {
		/* error ... attempt to resume parsing */
		switch Errflag {
		case 0: /* brand new error */
			exprlex.Error(exprErrorMessage(exprstate, exprtoken))
			Nerrs++
			if exprDebug >= 1 {
				__yyfmt__.Printf("%s", exprStatname(exprstate))
				__yyfmt__.Printf(" saw %s\n", exprTokname(exprtoken))
			}
			fallthrough

		case 1, 2: /* incompletely recovered error ... try again */
			Errflag = 3

			/* find a state where "error" is a legal shift action */
			for exprp >= 0 {
				exprn = exprPact[exprS[exprp].yys] + exprErrCode
				if exprn >= 0 && exprn < exprLast {
					exprstate = exprAct[exprn] /* simulate a shift of "error" */
					if exprChk[exprstate] == exprErrCode {
						goto exprstack
					}
				}

				/* the current p has no shift on "error", pop stack */
				if exprDebug >= 2 {
					__yyfmt__.Printf("error recovery pops state %d\n", exprS[exprp].yys)
				}
				exprp--
			}
			/* there is no state on the stack with an error shift ... abort */
			goto ret1

		case 3: /* no shift yet; clobber input char */
			if exprDebug >= 2 {
				__yyfmt__.Printf("error recovery discards %s\n", exprTokname(exprtoken))
			}
			if exprtoken == exprEofCode {
				goto ret1
			}
			exprrcvr.char = -1
			exprtoken = -1
			goto exprnewstate /* try again in the same state */
		}
	}

	/* reduction by production exprn */
	if exprDebug >= 2 {
		__yyfmt__.Printf("reduce %v in:\n\t%v\n", exprn, exprStatname(exprstate))
	}

	exprnt := exprn
	exprpt := exprp
	_ = exprpt // guard against "declared and not used"

	exprp -= exprR2[exprn]
	// exprp is now the index of $0. Perform the default action. Iff the
	// reduced production is ε, $1 is possibly out of range.
	if exprp+1 >= len(exprS) {
		nyys := make([]exprSymType, len(exprS)*2)
		copy(nyys, exprS)
		exprS = nyys
	}
	exprVAL = exprS[exprp+1]

	/* consult goto table to find next state */
	exprn = exprR1[exprn]
	exprg := exprPgo[exprn]
	exprj := exprg + exprS[exprp].yys + 1

	if exprj >= exprLast {
		exprstate = exprAct[exprg]
	} else {
		exprstate = exprAct[exprj]
		if exprChk[exprstate] != -exprn {
			exprstate = exprAct[exprg]
		}
	}
	// dummy call; replaced with literal code
	switch exprnt {

	case 1:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:77
		{
			exprlex.(*matExprLexer).parsedExpression = exprDollar[1].node
		}
	case 2:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:79
		{
			exprlex.(*matExprLexer).parsedExpression = exprDollar[1].node
		}
	case 3:
		exprDollar = exprS[exprpt-4 : exprpt+1]
		//line material_expr.y:82
		{
			exprVAL.node = BxdfNode{
				Type:       bxdfTypeFromName(exprDollar[1].sVal),
				Parameters: exprDollar[3].node.(BxdfParameterList),
			}
		}
	case 10:
		exprDollar = exprS[exprpt-0 : exprpt+1]
		//line material_expr.y:97
		{
			exprVAL.node = make(BxdfParameterList, 0)
		}
	case 12:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:101
		{
			exprVAL.node = BxdfParameterList{exprDollar[1].node.(BxdfParamNode)}
		}
	case 13:
		exprDollar = exprS[exprpt-3 : exprpt+1]
		//line material_expr.y:103
		{
			exprVAL.node = append(exprDollar[1].node.(BxdfParameterList), exprDollar[3].node.(BxdfParamNode))
		}
	case 14:
		exprDollar = exprS[exprpt-3 : exprpt+1]
		//line material_expr.y:106
		{
			exprVAL.node = BxdfParamNode{Name: exprDollar[1].sVal, Value: exprDollar[3].node}
		}
	case 15:
		exprDollar = exprS[exprpt-3 : exprpt+1]
		//line material_expr.y:108
		{
			exprVAL.node = BxdfParamNode{Name: exprDollar[1].sVal, Value: exprDollar[3].node}
		}
	case 16:
		exprDollar = exprS[exprpt-3 : exprpt+1]
		//line material_expr.y:110
		{
			exprVAL.node = BxdfParamNode{Name: exprDollar[1].sVal, Value: exprDollar[3].node}
		}
	case 17:
		exprDollar = exprS[exprpt-3 : exprpt+1]
		//line material_expr.y:112
		{
			exprVAL.node = BxdfParamNode{Name: exprDollar[1].sVal, Value: exprDollar[3].node}
		}
	case 18:
		exprDollar = exprS[exprpt-3 : exprpt+1]
		//line material_expr.y:114
		{
			exprVAL.node = BxdfParamNode{Name: exprDollar[1].sVal, Value: exprDollar[3].node}
		}
	case 19:
		exprDollar = exprS[exprpt-3 : exprpt+1]
		//line material_expr.y:116
		{
			exprVAL.node = BxdfParamNode{Name: exprDollar[1].sVal, Value: exprDollar[3].node}
		}
	case 20:
		exprDollar = exprS[exprpt-3 : exprpt+1]
		//line material_expr.y:118
		{
			exprVAL.node = BxdfParamNode{Name: exprDollar[1].sVal, Value: FloatNode(exprDollar[3].fVal)}
		}
	case 21:
		exprDollar = exprS[exprpt-3 : exprpt+1]
		//line material_expr.y:120
		{
			exprVAL.node = BxdfParamNode{Name: exprDollar[1].sVal, Value: exprDollar[3].node}
		}
	case 23:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:123
		{
			exprVAL.node = TextureNode(exprDollar[1].sVal)
		}
	case 24:
		exprDollar = exprS[exprpt-7 : exprpt+1]
		//line material_expr.y:126
		{
			exprVAL.node = Vec3Node{exprDollar[2].fVal, exprDollar[4].fVal, exprDollar[6].fVal}
		}
	case 25:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:128
		{
			exprVAL.node = FloatNode(exprDollar[1].fVal)
		}
	case 26:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:129
		{
			exprVAL.node = MaterialNameNode(exprDollar[1].sVal)
		}
	case 27:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:131
		{
			exprVAL.node = FloatNode(exprDollar[1].fVal)
		}
	case 28:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:132
		{
			exprVAL.node = TextureNode(exprDollar[1].sVal)
		}
	case 29:
		exprDollar = exprS[exprpt-10 : exprpt+1]
		//line material_expr.y:135
		{
			exprVAL.node = MixNode{
				Expressions: [2]ExprNode{exprDollar[3].node, exprDollar[5].node},
				Weights:     [2]float32{exprDollar[7].fVal, exprDollar[9].fVal},
			}
		}
	case 30:
		exprDollar = exprS[exprpt-8 : exprpt+1]
		//line material_expr.y:142
		{
			exprVAL.node = MixMapNode{
				Expressions: [2]ExprNode{exprDollar[3].node, exprDollar[5].node},
				Texture:     TextureNode(exprDollar[7].sVal),
			}
		}
	case 31:
		exprDollar = exprS[exprpt-6 : exprpt+1]
		//line material_expr.y:149
		{
			exprVAL.node = BumpMapNode{
				Expression: exprDollar[3].node,
				Texture:    TextureNode(exprDollar[5].sVal),
			}
		}
	case 32:
		exprDollar = exprS[exprpt-6 : exprpt+1]
		//line material_expr.y:156
		{
			exprVAL.node = NormalMapNode{
				Expression: exprDollar[3].node,
				Texture:    TextureNode(exprDollar[5].sVal),
			}
		}
	case 33:
		exprDollar = exprS[exprpt-12 : exprpt+1]
		//line material_expr.y:163
		{
			exprVAL.node = DisperseNode{
				Expression: exprDollar[3].node,
				IntIOR:     exprDollar[7].node.(Vec3Node),
				ExtIOR:     exprDollar[11].node.(Vec3Node),
			}
		}
	case 36:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:174
		{
			exprVAL.node = MaterialRefNode(exprDollar[1].sVal)
		}
	}
	goto exprstack /* stack new state and value */
}
