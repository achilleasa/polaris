#ifndef CONSTANTS_CL
#define CONSTANTS_CL

// Re-define useful constants (from math.h) as floats

#define C_E            2.7182818284590452354f   /* e */
#define C_LOG2E        1.4426950408889634074f   /* log_2 e */
#define C_LOG10E       0.43429448190325182765f  /* log_10 e */
#define C_LN2          0.69314718055994530942f  /* log_e 2 */
#define C_LN10         2.30258509299404568402f  /* log_e 10 */
#define C_PI           3.14159265358979323846f  /* pi */
#define C_TWO_TIMES_PI 6.28318530718f           /* 2pi */
#define C_PI_2         1.57079632679489661923f  /* pi/2 */
#define C_PI_4         0.78539816339744830962f  /* pi/4 */
#define C_1_PI         0.31830988618379067154f  /* 1/pi */
#define C_1_TWO_TIMES_PI 0.15915494309f         /* 1/2pi */
#define C_2_PI         0.63661977236758134308f  /* 2/pi */
#define C_2_SQRTPI     1.12837916709551257390f  /* 2/sqrt(pi) */
#define C_SQRT2        1.41421356237309504880f  /* sqrt(2) */
#define C_SQRT1_2      0.70710678118654752440f  /* 1/sqrt(2) */

// Intersection constants
#define INTERSECTION_EPSILON 0.00001f
#define INTERSECTION_EPSILON_X2 (INTERSECTION_EPSILON * 2.0f)

#endif
