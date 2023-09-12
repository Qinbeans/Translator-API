import jax; jax.config.update('jax_platforms', 'cpu'); jax.config.update('jax_default_matmul_precision', jax.lax.Precision.HIGHEST)

import jax.numpy as np
from transformers import BartConfig, BartTokenizer, BertTokenizer
import sys

from TransCan.lib.Generator import Generator
from TransCan.lib.param_utils.load_params import load_params
from TransCan.lib.en_kfw_nmt.fwd_transformer_encoder_part import fwd_transformer_encoder_part

class Translator:
    def __init__(self) -> None:
        params = load_params(sys.argv[1])
        params = jax.tree_map(np.asarray, params)

        self.tokenizer_en = BartTokenizer.from_pretrained('facebook/bart-base')
        self.tokenizer_yue = BertTokenizer.from_pretrained('Ayaka/bart-base-cantonese')

        config = BartConfig.from_pretrained('Ayaka/bart-base-cantonese')
        self.generator = Generator({'embedding': params['decoder_embedding'], **params}, config=config)
        self.params = params
    def inference(self, input: str) -> str:
        inputs = self.tokenizer_en([input], return_tensors='jax', padding=True)
        src = inputs.input_ids.astype(np.uint16)
        mask_enc_1d = inputs.attention_mask.astype(np.bool_)
        mask_enc = np.einsum('bi,bj->bij', mask_enc_1d, mask_enc_1d)[:, None]

        encoder_last_hidden_output = fwd_transformer_encoder_part(self.params, src, mask_enc)
        generate_ids = self.generator.generate(encoder_last_hidden_output, mask_enc_1d, num_beams=5, max_length=128)

        decoded_sentences = self.tokenizer_yue.batch_decode(generate_ids, skip_special_tokens=True, clean_up_tokenization_spaces=False)
        return decoded_sentences[0]
