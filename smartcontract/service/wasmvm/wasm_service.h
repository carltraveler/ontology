#include<stdint.h>

typedef struct InterOpCtx {
	uint32_t		height;
	uint8_t*		bloc_khash;
	uint64_t		timestamp;
	uint8_t*		tx_hash;
	uint8_t*		self_address;
	uint8_t*		callers;
	size_t			callers_num;
	uint8_t*		witness;
	size_t			witness_num;
	uint8_t*		input;
	size_t			input_len;
	uint64_t		gas_left;
	uint8_t*		call_output;
	size_t			call_output_len;
} InterOpCtx;


void ontio_call_invoke(uint8_t *code, uint32_t codelen, InterOpCtx ctx);
