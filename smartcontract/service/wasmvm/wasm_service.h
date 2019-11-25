#ifndef WASM_SERVICE_H
#define WASM_SERVICE_H
typedef unsigned long long uint64_t;
typedef unsigned int uint32_t;
typedef unsigned char uint8_t;
typedef unsigned long size_t;

typedef struct {
	uint32_t			height;
	uint8_t*			block_hash;
	uint64_t			timestamp;
	uint8_t*			tx_hash;
	uint8_t*			self_address;
	uint8_t*			callers;
	size_t				callers_num;
	uint8_t*			witness;
	size_t				witness_num;
	uint8_t*			input;
	size_t				input_len;
	uint64_t			wasmvm_service_ptr;
	uint64_t			gas_left;
} InterOpCtx;

typedef struct {
	uint32_t err;
	uint8_t* errmsg;
} Cgoerror;

typedef struct {
	uint8_t* output;
	uint32_t outputlen;
	uint32_t err;
	uint8_t* errmsg;
} Cgooutput;

typedef struct {
	uint64_t v;
	uint32_t err;
	uint8_t* errmsg;
} Cgou64;

typedef Cgooutput Cgobuffer;

Cgooutput ontio_call_invoke(uint8_t *code, uint32_t codelen, InterOpCtx ctx);
void ontio_free_cgooutput(Cgooutput output);
Cgobuffer ontio_read_wasmvm_memory(uint8_t* vmctx, uint32_t data_ptr, uint32_t l);
void ontio_memfree(uint8_t* mem);
Cgoerror ontio_error(uint8_t *ptr, uint32_t len);
Cgou64 ontio_wasm_service_ptr(uint8_t *vmctx);
Cgoerror ontio_set_calloutput(uint8_t *vmctx, uint8_t* buff, uint32_t size);
#endif
