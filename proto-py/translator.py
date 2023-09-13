import jax; jax.config.update('jax_platforms', 'cpu'); jax.config.update('jax_default_matmul_precision', jax.lax.Precision.HIGHEST)

import jax.numpy as np
from transformers import BartConfig, BartTokenizer, BertTokenizer
import grpc

from TransCan.lib.Generator import Generator
from TransCan.lib.param_utils.load_params import load_params
from TransCan.lib.en_kfw_nmt.fwd_transformer_encoder_part import fwd_transformer_encoder_part

from translate_pb2 import TranslateRequest, TranslateResponse
from translate_pb2_grpc import TranslatorServicer

import logging

class Translator(TranslatorServicer):
    def __init__(self, param: str) -> None:
        '''
        Initialize the translator
        '''
        logging.debug('Initializing Translator\n\tLoading model parameters')
        params = load_params(param)
        params = jax.tree_map(np.asarray, params)

        logging.debug('\tLoading tokenizer')
        self.tokenizer_en = BartTokenizer.from_pretrained('facebook/bart-base')
        self.tokenizer_yue = BertTokenizer.from_pretrained('Ayaka/bart-base-cantonese')

        logging.debug('\tLoading model')
        config = BartConfig.from_pretrained('Ayaka/bart-base-cantonese')
        self.generator = Generator({'embedding': params['decoder_embedding'], **params}, config=config)
        self.params = params

        logging.debug('Done')

    def Translate(self, _input: TranslateRequest, context: grpc.ServicerContext) -> TranslateResponse:
        try:
            logging.debug('Translating: ' + _input.text)

            logging.debug('\tTokenizing')
            inputs = self.tokenizer_en([_input.text], return_tensors='jax', padding=True)
            src = inputs.input_ids.astype(np.uint16)
            mask_enc_1d = inputs.attention_mask.astype(np.bool_)
            mask_enc = np.einsum('bi,bj->bij', mask_enc_1d, mask_enc_1d)[:, None]

            logging.debug('\tTranslating')
            encoder_last_hidden_output = fwd_transformer_encoder_part(self.params, src, mask_enc)
            generate_ids = self.generator.generate(encoder_last_hidden_output, mask_enc_1d, num_beams=5, max_length=128)

            logging.debug('\tDecoding')
            decoded_sentences = self.tokenizer_yue.batch_decode(generate_ids, skip_special_tokens=True, clean_up_tokenization_spaces=False)
            logging.debug('Done')
            return TranslateResponse(text=decoded_sentences[0], details=_input.details)
        except Exception as e:
            logging.error(e)
            details = _input.details
            details.message = "Error: " + str(e)
            return TranslateResponse(text='', details=details)
