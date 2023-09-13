# Start protobuf server

import logging
from concurrent.futures import ThreadPoolExecutor

import grpc
import os

from translate_pb2_grpc import add_TranslatorServicer_to_server
from translator import Translator

import argparse

def get_args() -> argparse.Namespace:
    parser = argparse.ArgumentParser(description='Translator server')
    parser.add_argument('--param', '-P', type=str, default=os.path.join(os.path.dirname(__file__), 'data', 'atomic-thunder-15-7.dat'), help='Path to model parameters')
    parser.add_argument('--port','-p', type=int, default=50051, help='Port to listen on')
    parser.add_argument('--host', '-H', type=str, default='localhost', help='Host to listen on')
    parser.add_argument('--connect', '-c', type=str, default='localhost:50051', help='Host to connect to')
    parser.add_argument('--debug', '-v', action='store_true', help='Enable debug logging')
    return parser.parse_args()

def main() -> None:
    args = get_args()

    if args.debug:
        logging.basicConfig(level=logging.DEBUG)
        logging.info('Debug logging enabled')
    else:
        logging.basicConfig(level=logging.INFO)
        logging.info('Debug logging disabled')

    logging.info('Starting server')

    host = args.host
    port = args.port

    if args.connect:
        host, port = args.connect.split(':')
        port = int(port)

    logging.debug('Connecting to ' + host + ':' + str(port))

    server = grpc.server(ThreadPoolExecutor())
    add_TranslatorServicer_to_server(Translator(args.param), server)

    server.add_insecure_port(host + ':' + str(port))
    server.start()

    logging.info('Server started on ' + host + ':' + str(port))
    server.wait_for_termination()

if __name__ == '__main__':
    main()