%{
//go:generate go tool yacc -o material_expr.y.go -p expr material_expr.y
package compiler

import (
	"fmt"
	"bytes"
	"strconv"
	"unicode/utf8"

	"github.com/achilleasa/go-pathtrace/scene"
)

%}

%union {
	num float32
	nodeId uint32

	compiler *sceneCompiler
	material *scene.ParsedMaterial
}

%type <nodeId> BSDF
%type <nodeId> blend_func
%type <nodeId> blend_operand

%token '(' ')' ','
%token	<num> NUM

%token <nodeId> MIX
%token <nodeId> FRESNEL

%token <nodeId> DIFFUSE
%token <nodeId> SPECULAR_REFLECTION
%token <nodeId> SPECULAR_TRANSMISSION
%token <nodeId> SPECULAR_MICROFACET
%token <nodeId> EMISSIVE

%%

top:
	BSDF
|	blend_func

blend_func:
	MIX '(' blend_operand ',' blend_operand ',' NUM ')'
	{
		node := &scene.MaterialNode{}
		node.Init()
		node.IsNode = 1
		node.SetLeftIndex($3)
		node.SetRightIndex($5)
		node.SetBlendFunc(scene.Mix)
		node.Nval = $7
		node.SetNvalTex(exprVAL.material.NiTex)
		$$ = exprVAL.compiler.appendMaterialNode(node)
	}
|	FRESNEL '(' blend_operand ',' blend_operand ')'
	{
		node := &scene.MaterialNode{}
		node.Init()
		node.IsNode = 1
		node.SetLeftIndex($3)
		node.SetRightIndex($5)
		node.SetBlendFunc(scene.Fresnel)
		node.Nval = exprVAL.material.Ni
		node.SetNvalTex(exprVAL.material.NiTex)
		$$ = exprVAL.compiler.appendMaterialNode(node)
	}

blend_operand:
	BSDF
|	blend_func

BSDF:
	DIFFUSE
	{
		node := &scene.MaterialNode{}
		node.Init()
		node.Kval = exprVAL.material.Kd.Vec4(0)
		node.SetKvalTex(exprVAL.material.KdTex)
		node.SetNormalTex(exprVAL.material.NormalTex)
		node.SetBxdfType(scene.Diffuse)
		$$ = exprVAL.compiler.appendMaterialNode(node)
	}
|	SPECULAR_REFLECTION
	{
		node := &scene.MaterialNode{}
		node.Init()
		node.Kval = exprVAL.material.Ks.Vec4(0)
		node.SetKvalTex(exprVAL.material.KsTex)
		node.SetNormalTex(exprVAL.material.NormalTex)
		node.SetBxdfType(scene.SpecularReflection)
		$$ = exprVAL.compiler.appendMaterialNode(node)
	}
|	SPECULAR_TRANSMISSION
	{
		node := &scene.MaterialNode{}
		node.Init()
		node.Kval = exprVAL.material.Tf.Vec4(0)
		node.SetKvalTex(exprVAL.material.TfTex)
		node.SetNormalTex(exprVAL.material.NormalTex)
		node.Nval = exprVAL.material.Ni
		node.SetNvalTex(exprVAL.material.NiTex)
		node.SetBxdfType(scene.SpecularTransmission)
		$$ = exprVAL.compiler.appendMaterialNode(node)
	}
|	SPECULAR_MICROFACET
	{
		node := &scene.MaterialNode{}
		node.Init()
		node.Kval = exprVAL.material.Ks.Vec4(0)
		node.SetKvalTex(exprVAL.material.KsTex)
		node.SetNormalTex(exprVAL.material.NormalTex)
		node.Nval = exprVAL.material.Nr
		node.SetNvalTex(exprVAL.material.NrTex)
		node.SetBxdfType(scene.SpecularMicrofacet)
		$$ = exprVAL.compiler.appendMaterialNode(node)
	}
|	EMISSIVE
	{
		node := &scene.MaterialNode{}
		node.Init()
		node.Kval = exprVAL.material.Ke.Vec4(0)
		node.SetKvalTex(exprVAL.material.KeTex)
		node.SetNormalTex(exprVAL.material.NormalTex)
		node.SetBxdfType(scene.Emissive)
		$$ = exprVAL.compiler.appendMaterialNode(node)
	}
%%

// The parser expects the lexer to return 0 on EOF.
const EOF = 0

type matNode struct {
	ntype int
	left, right *matNode
}

// The parser uses the type <prefix>Lex as a lexer.  It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type exprLex struct {
	line []byte
	peek rune
	tokenBuf bytes.Buffer

	compiler *sceneCompiler
	material *scene.ParsedMaterial
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
		if (c >= 'a' && c <= 'z') || (c>= 'A' && c <= 'Z') || (c=='_' || c== '-'){
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
	case "D": return DIFFUSE
	case "S": return SPECULAR_REFLECTION
	case "T": return SPECULAR_TRANSMISSION
	case "M": return SPECULAR_MICROFACET
	case "E": return EMISSIVE
	case "mix": return MIX
	case "fresnel": return FRESNEL
	default:
		x.Error(fmt.Sprintf("invalid expression %q", val))
		return EOF
	}
}

// Append a rune to the toke buffer.
func (x *exprLex) captureRune(c rune){
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

