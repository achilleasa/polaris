//line material_expr.y:2

//go:generate go tool yacc -o material_expr.y.go -p expr material_expr.y
package compiler

import __yyfmt__ "fmt"

//line material_expr.y:3
import (
	"bytes"
	"fmt"
	"strconv"
	"unicode/utf8"

	"github.com/achilleasa/go-pathtrace/scene"
)

//line material_expr.y:16
type exprSymType struct {
	yys    int
	num    float32
	nodeId uint32

	compiler *sceneCompiler
	material *scene.ParsedMaterial
}

const NUM = 57346
const MIX = 57347
const FRESNEL = 57348
const DIFFUSE = 57349
const SPECULAR_REFLECTION = 57350
const SPECULAR_TRANSMISSION = 57351
const SPECULAR_MICROFACET = 57352
const EMISSIVE = 57353

var exprToknames = [...]string{
	"$end",
	"error",
	"$unk",
	"'('",
	"')'",
	"','",
	"NUM",
	"MIX",
	"FRESNEL",
	"DIFFUSE",
	"SPECULAR_REFLECTION",
	"SPECULAR_TRANSMISSION",
	"SPECULAR_MICROFACET",
	"EMISSIVE",
}
var exprStatenames = [...]string{}

const exprEofCode = 1
const exprErrCode = 2
const exprInitialStackSize = 16

//line material_expr.y:139

// The parser expects the lexer to return 0 on EOF.
const EOF = 0

type matNode struct {
	ntype       int
	left, right *matNode
}

// The parser uses the type <prefix>Lex as a lexer.  It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type exprLex struct {
	line     []byte
	peek     rune
	tokenBuf bytes.Buffer

	compiler  *sceneCompiler
	material  *scene.ParsedMaterial
	lastError error
}

// The parser calls this method to get each new token.
func (x *exprLex) Lex(yylval *exprSymType) int {
	// Pass compiler/material references to symbol
	yylval.compiler = x.compiler
	yylval.material = x.material

	for {
		c := x.next()
		switch c {
		case EOF:
			return EOF
		case '0', '1', '2', '3', '4', '5', '6', '7', '8', '9':
			x.tokenBuf.Reset()
			return x.lexFloat32(c, yylval)
		case '(', ')', ',':
			return int(c)
		case ' ', '\t', '\n', '\r':
		default:
			x.tokenBuf.Reset()
			return x.lexText(c, yylval)
		}
	}
}

// Lex a float.
func (x *exprLex) lexFloat32(c rune, yylval *exprSymType) int {
	x.captureRune(c)
	for {
		c = x.next()
		if (c >= '0' && c <= '9') || (c == '.' || c == 'e' || c == 'E') {
			x.captureRune(c)
			continue
		}
		break
	}

	if c != EOF {
		x.peek = c
	}

	val, err := strconv.ParseFloat(x.tokenBuf.String(), 32)
	if err != nil {
		x.Error(fmt.Sprintf("invalid float value %q: %v", x.tokenBuf.String(), err))
		return EOF
	}

	yylval.num = float32(val)
	return NUM
}

// Lex a text token.
func (x *exprLex) lexText(c rune, yylval *exprSymType) int {
	x.captureRune(c)
	for {
		c = x.next()
		if (c >= 'a' && c <= 'z') || (c >= 'A' && c <= 'Z') || (c == '_' || c == '-') {
			x.captureRune(c)
			continue
		}
		break
	}

	if c != EOF {
		x.peek = c
	}

	val := x.tokenBuf.String()

	switch val {
	case "D":
		return DIFFUSE
	case "S":
		return SPECULAR_REFLECTION
	case "T":
		return SPECULAR_TRANSMISSION
	case "M":
		return SPECULAR_MICROFACET
	case "E":
		return EMISSIVE
	case "mix":
		return MIX
	case "fresnel":
		return FRESNEL
	default:
		x.Error(fmt.Sprintf("invalid expression %q", val))
		return EOF
	}
}

// Append a rune to the toke buffer.
func (x *exprLex) captureRune(c rune) {
	x.tokenBuf.WriteRune(c)
}

// Return the next rune for the lexer.
func (x *exprLex) next() rune {
	if x.peek != EOF {
		r := x.peek
		x.peek = EOF
		return r
	}
	if len(x.line) == 0 {
		return EOF
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
func (x *exprLex) Error(s string) {
	// Keep the first error we encountered
	if x.lastError == nil {
		x.lastError = fmt.Errorf(s)
	}
}

//line yacctab:1
var exprExca = [...]int{
	-1, 1,
	1, -1,
	-2, 0,
}

const exprNprod = 12
const exprPrivate = 57344

var exprTokenNames []string
var exprStates []string

const exprLast = 24

var exprAct = [...]int{

	13, 9, 10, 4, 5, 6, 7, 8, 23, 21,
	18, 17, 24, 16, 22, 12, 11, 1, 19, 20,
	15, 3, 14, 2,
}
var exprPact = [...]int{

	-7, -1000, -1000, -1000, -1000, -1000, -1000, -1000, -1000, 12,
	11, -7, -7, 5, -1000, -1000, 4, -7, -7, 3,
	9, 1, -1000, 7, -1000,
}
var exprPgo = [...]int{

	0, 22, 20, 0, 17,
}
var exprR1 = [...]int{

	0, 4, 4, 2, 2, 3, 3, 1, 1, 1,
	1, 1,
}
var exprR2 = [...]int{

	0, 1, 1, 8, 6, 1, 1, 1, 1, 1,
	1, 1,
}
var exprChk = [...]int{

	-1000, -4, -1, -2, 10, 11, 12, 13, 14, 8,
	9, 4, 4, -3, -1, -2, -3, 6, 6, -3,
	-3, 6, 5, 7, 5,
}
var exprDef = [...]int{

	0, -2, 1, 2, 7, 8, 9, 10, 11, 0,
	0, 0, 0, 0, 5, 6, 0, 0, 0, 0,
	0, 0, 4, 0, 3,
}
var exprTok1 = [...]int{

	1, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	3, 3, 3, 3, 3, 3, 3, 3, 3, 3,
	4, 5, 3, 3, 6,
}
var exprTok2 = [...]int{

	2, 3, 7, 8, 9, 10, 11, 12, 13, 14,
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

	case 3:
		exprDollar = exprS[exprpt-8 : exprpt+1]
		//line material_expr.y:48
		{
			node := &scene.MaterialNode{}
			node.Init()
			node.IsNode = 1
			node.SetLeftIndex(exprDollar[3].nodeId)
			node.SetRightIndex(exprDollar[5].nodeId)
			node.SetBlendFunc(scene.Mix)
			node.Nval = exprDollar[7].num
			node.SetNvalTex(exprVAL.material.NiTex)
			exprVAL.nodeId = exprVAL.compiler.appendMaterialNode(node)
		}
	case 4:
		exprDollar = exprS[exprpt-6 : exprpt+1]
		//line material_expr.y:60
		{
			node := &scene.MaterialNode{}
			node.Init()
			node.IsNode = 1
			node.SetLeftIndex(exprDollar[3].nodeId)
			node.SetRightIndex(exprDollar[5].nodeId)
			node.SetBlendFunc(scene.Fresnel)
			node.Nval = exprVAL.material.Ni
			node.SetNvalTex(exprVAL.material.NiTex)
			exprVAL.nodeId = exprVAL.compiler.appendMaterialNode(node)
		}
	case 7:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:78
		{
			node := &scene.MaterialNode{}
			node.Init()
			node.Kval = exprVAL.material.Kd.Vec4(0)
			node.SetKvalTex(exprVAL.material.KdTex)
			node.SetNormalTex(exprVAL.material.NormalTex)
			node.SetBxdfType(scene.Diffuse)
			exprVAL.nodeId = exprVAL.compiler.appendMaterialNode(node)
		}
	case 8:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:88
		{
			node := &scene.MaterialNode{}
			node.Init()
			node.Kval = exprVAL.material.Ks.Vec4(0)
			node.SetKvalTex(exprVAL.material.KsTex)
			node.SetNormalTex(exprVAL.material.NormalTex)
			node.SetBxdfType(scene.SpecularReflection)
			exprVAL.nodeId = exprVAL.compiler.appendMaterialNode(node)
		}
	case 9:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:98
		{
			node := &scene.MaterialNode{}
			node.Init()
			node.Kval = exprVAL.material.Tf.Vec4(0)
			node.SetKvalTex(exprVAL.material.TfTex)
			node.SetNormalTex(exprVAL.material.NormalTex)
			if exprVAL.material.Ni <= 0.0 {
				exprVAL.material.Ni = 1.0
			}
			node.IOR = exprVAL.material.Ni
			node.SetIORTex(exprVAL.material.NiTex)
			node.SetBxdfType(scene.SpecularTransmission)
			exprVAL.nodeId = exprVAL.compiler.appendMaterialNode(node)
		}
	case 10:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:113
		{
			node := &scene.MaterialNode{}
			node.Init()
			node.Kval = exprVAL.material.Ks.Vec4(0)
			node.SetKvalTex(exprVAL.material.KsTex)
			node.SetNormalTex(exprVAL.material.NormalTex)
			node.Nval = exprVAL.material.Nr
			node.SetNvalTex(exprVAL.material.NrTex)
			if exprVAL.material.Ni <= 0.0 {
				exprVAL.material.Ni = 1.0
			}
			node.IOR = exprVAL.material.Ni
			node.SetIORTex(exprVAL.material.NiTex)
			node.SetBxdfType(scene.SpecularMicrofacet)
			exprVAL.nodeId = exprVAL.compiler.appendMaterialNode(node)
		}
	case 11:
		exprDollar = exprS[exprpt-1 : exprpt+1]
		//line material_expr.y:130
		{
			node := &scene.MaterialNode{}
			node.Init()
			node.Kval = exprVAL.material.Ke.Vec4(0)
			node.SetKvalTex(exprVAL.material.KeTex)
			node.SetNormalTex(exprVAL.material.NormalTex)
			node.SetBxdfType(scene.Emissive)
			exprVAL.nodeId = exprVAL.compiler.appendMaterialNode(node)
		}
	}
	goto exprstack /* stack new state and value */
}
