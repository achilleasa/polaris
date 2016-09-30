%{
//go:generate go tool yacc -o material_expr.y.go -p expr material_expr.y
package material

import (
	"fmt"
	"bytes"
	"strconv"
	"unicode/utf8"
)

%}

%union {
	cVal rune
	fVal float32
	sVal string
	node ExprNode
}

/* basic entities */
%token <cVal> tokLPAREN
%token <cVal> tokRPAREN
%token <cVal> tokLCURLY
%token <cVal> tokRCURLY
%token <cVal> tokCOMMA
%token <cVal> tokCOLON

%token <fVal> tokFLOAT
%token <sVal> tokMATERIAL_NAME   /* string enclosed in quotes */
%token <sVal> tokTEXTURE	/* texture filename */

/* tokParameter names */
%token <sVal> tokREFLECTANCE
%token <sVal> tokSPECULARITY
%token <sVal> tokTRANSMITTANCE
%token <sVal> tokRADIANCE
%token <sVal> tokINT_IOR
%token <sVal> tokEXT_IOR
%token <sVal> tokSCALE 
%token <sVal> tokROUGHNESS

/* tokBxDF types */
%token <sVal> tokDIFFUSE 
%token <sVal> tokCONDUCTOR
%token <sVal> tokROUGH_CONDUCTOR
%token <sVal> tokDIELECTRIC
%token <sVal> tokROUGH_DIELECTRIC
%token <sVal> tokEMISSIVE 

/* tokBlend functions */
%token <sVal> tokMIX
%token <sVal> tokMIX_MAP
%token <sVal> tokBUMP_MAP
%token <sVal> tokNORMAL_MAP
%token <sVal> tokDISPERSE

/* types for non-token items */
%type <node> material_def
%type <node> float3
%type <node> bxdf_parameter
%type <node> float3_or_texture
%type <node> float_or_name
%type <node> float_or_texture
%type <node> op_spec
%type <node> opt_bxdf_parameter_list
%type <node> bxdf_parameter_list
%type <node> bxdf_spec
%type <node> bxdf_or_op_spec
%type <sVal> bxdf_type

/* rule entry point */
%start material_def

%%
material_def: bxdf_spec 
	    { exprlex.(*matExprLexer).parsedExpression = $1 }
	    | op_spec
	    { exprlex.(*matExprLexer).parsedExpression = $1 } 

bxdf_spec: bxdf_type tokLPAREN opt_bxdf_parameter_list tokRPAREN
	 { 
	 	$$ = BxdfNode {
			Type: bxdfTypeFromName($1),
			Parameters: $3.(BxdfParameterList),
		}
	}

bxdf_type: tokDIFFUSE
	 | tokCONDUCTOR
	 | tokROUGH_CONDUCTOR
	 | tokDIELECTRIC
	 | tokROUGH_DIELECTRIC
	 | tokEMISSIVE

opt_bxdf_parameter_list: /* empty */
		       { $$ = make(BxdfParameterList, 0) }
		       | bxdf_parameter_list

bxdf_parameter_list: bxdf_parameter
		   { $$ = BxdfParameterList{$1.(BxdfParamNode)} }
	           | bxdf_parameter_list tokCOMMA bxdf_parameter
		   { $$ = append($1.(BxdfParameterList), $3.(BxdfParamNode)) }

bxdf_parameter: tokREFLECTANCE tokCOLON float3_or_texture
	      { $$ = BxdfParamNode{Name: $1, Value: $3} }
	      | tokSPECULARITY tokCOLON float3_or_texture
	      { $$ = BxdfParamNode{Name: $1, Value: $3} }
	      | tokTRANSMITTANCE tokCOLON float3_or_texture
	      { $$ = BxdfParamNode{Name: $1, Value: $3} }
	      | tokRADIANCE tokCOLON float3_or_texture
	      { $$ = BxdfParamNode{Name: $1, Value: $3} }
	      | tokINT_IOR tokCOLON float_or_name
	      { $$ = BxdfParamNode{Name: $1, Value: $3} }
	      | tokEXT_IOR tokCOLON float_or_name
	      { $$ = BxdfParamNode{Name: $1, Value: $3} }
	      | tokSCALE tokCOLON tokFLOAT
	      { $$ = BxdfParamNode{Name: $1, Value: FloatNode($3)} }
	      | tokROUGHNESS tokCOLON float_or_texture
	      { $$ = BxdfParamNode{Name: $1, Value: $3} }

float3_or_texture: float3
		 | tokTEXTURE { $$ = TextureNode($1) }

float3: tokLCURLY tokFLOAT tokCOMMA tokFLOAT tokCOMMA tokFLOAT tokRCURLY 
      	{ $$ = Vec3Node{$2, $4, $6} }

float_or_name: tokFLOAT { $$ = FloatNode($1) }
 	     | tokMATERIAL_NAME { $$ = MaterialNameNode($1) }

float_or_texture: tokFLOAT { $$ = FloatNode($1) }
		| tokTEXTURE { $$ = TextureNode($1) }

op_spec: tokMIX tokLPAREN bxdf_or_op_spec tokCOMMA bxdf_or_op_spec tokCOMMA tokFLOAT tokCOMMA tokFLOAT tokRPAREN
	  { 
	  	$$ = MixNode{ 
	  		Expressions: [2]ExprNode{$3, $5},
			Weights: [2]float32{$7, $9},
		}
	  }
	  | tokMIX_MAP tokLPAREN bxdf_or_op_spec tokCOMMA bxdf_or_op_spec tokCOMMA tokTEXTURE tokRPAREN
	  { 
	  	$$ = MixMapNode{ 
	  		Expressions: [2]ExprNode{$3, $5},
			Texture: TextureNode($7),
		}
	  }
	  | tokBUMP_MAP tokLPAREN bxdf_or_op_spec tokCOMMA tokTEXTURE tokRPAREN
	  {
	  	$$ = BumpMapNode {
			Expression: $3,
			Texture: TextureNode($5),
		}
	  }
	  | tokNORMAL_MAP tokLPAREN bxdf_or_op_spec tokCOMMA tokTEXTURE tokRPAREN
	  {
	  	$$ = NormalMapNode {
			Expression: $3,
			Texture: TextureNode($5),
		}
	  }
	  | tokDISPERSE tokLPAREN bxdf_or_op_spec tokCOMMA tokINT_IOR tokCOLON float3 tokCOMMA tokEXT_IOR tokCOLON float3 tokRPAREN
	  {
	  	$$ = DisperseNode{
			Expression: $3,
			IntIOR: $7.(Vec3Node),
			ExtIOR: $11.(Vec3Node),
		}
	  }

bxdf_or_op_spec: bxdf_spec
	       | op_spec
	       | tokMATERIAL_NAME
	       {
	       	$$ = MaterialRefNode($1)
	       }
%%

// The parser expects the lexer to return 0 on kEOF.
const tokEOF = 0

// The parser uses the type <prefix>Lex as a lexer.  It must provide
// the methods Lex(*<prefix>SymType) int and Error(string).
type matExprLexer struct {
	line []byte
	peek rune
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
		if (c >= 'a' && c <= 'z') || (c>= 'A' && c <= 'Z') || c=='_'{
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
	case "diffuse": return tokDIFFUSE
	case "conductor": return tokCONDUCTOR
	case "roughConductor": return tokROUGH_CONDUCTOR
	case "dielectric": return tokDIELECTRIC
	case "roughDielectric": return tokROUGH_DIELECTRIC
	case "emissive": return tokEMISSIVE
	// Operators
	case "mix": return tokMIX
	case "mixMap": return tokMIX_MAP
	case "bumpMap": return tokBUMP_MAP
	case "normalMap": return tokNORMAL_MAP
	case "disperse": return tokDISPERSE
	// Parameters
	case ParamReflectance: return tokREFLECTANCE
	case ParamSpecularity: return tokSPECULARITY
	case ParamTransmittance: return tokTRANSMITTANCE
	case ParamRadiance: return tokRADIANCE
	case ParamIntIOR: return tokINT_IOR
	case ParamExtIOR: return tokEXT_IOR
	case ParamScale: return tokSCALE
	case ParamRoughness: return tokROUGHNESS
	default:
		x.Error(fmt.Sprintf("invalid expression %q", yylval.sVal))
		return tokEOF
	}
}

// Append a rune to the token buffer.
func (x *matExprLexer) captureRune(c rune){
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
	matLexer := &matExprLexer{ line : []byte(input) }
	exprNewParser().Parse(matLexer)
	if matLexer.lastError != nil {
		return nil, matLexer.lastError
	}

	return matLexer.parsedExpression, nil
}
