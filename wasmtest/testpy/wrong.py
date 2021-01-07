OntCversion = '2.0.0'

from ontology.interop.System.App import DynamicAppCall
from ontology.libont import hexstring2address

def Main(operation, args):
    if operation == 'mintToken':
        if len(args) != 3:
            return False
        player = args[0]
        tokenId = args[1]
        # textContext at last
        textContext = args[2]
        contract= textContext[0]["test2.avm"]
        return mintToken(player, contract, tokenId)
    elif operation == "testcase":
        return testcase()

    return False

def testcase():
    return '''
    [
        [{"needcontext":true,"env":{"witness":[]}, "method":"mintToken", "param":"[address:AbG3ZgFrMK6fqwXWR1WkQ1d1EYVunCwknu,int:2]", "expected":"int:1"}
        ]
    ]'''

def mintToken(player, p1, p2):
    assert (player == _ownerOf(p1, p2))
    return True

def _ownerOf(contract, tokenId):
    params = [tokenId]
    return DynamicAppCall(contract, 'ownerOf', params)
