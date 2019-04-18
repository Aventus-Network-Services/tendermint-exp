const axios = require('axios')
const Base64 = require('js-base64').Base64
const EthCrypto = require('eth-crypto')
const utils = require('web3-utils')
const node = 'http://0.0.0.0:26657/'
const sell = process.argv[2]

const addr1 = '0x91ea89ded135e9eea4386ae8f2a8a525afa05f7f'
const addr2 = '0x488184297bc674da394a8bf0eed703295cbace5c'
const privateKey = '1977d13b2337ac36005c63316e1771a8a2edce6cdcf169779bf0f91ac1ffe63d'

const sale = {
  id: 1,
  nonce: 1,
  details: "ticket",
  ownerAddr: addr1,
  prevOwnerProof: "0x"
}

const main = async () => {
  if (sell === 'sell') {
    const result = await axios.post(node, {
      "method": "broadcast_tx_sync",
      "jsonrpc": "2.0",
      "params": [ Base64.toBase64(JSON.stringify(sale)) ]
    })
    console.log(result.data)
  } else if (sell === 'resell') {
    const hash = utils.soliditySha3(sale.id, sale.nonce, sale.details, sale.ownerAddr, sale.prevOwnerProof)
    const prevOwnerProof = EthCrypto.sign(privateKey, hash)

    const resale = {
      id: 1,
      nonce: 2,
      details: "ticket",
      ownerAddr: addr2,
      prevOwnerProof
    }

    const result = await axios.post(node, {
      "method": "broadcast_tx_sync",
      "jsonrpc": "2.0",
      "params": [ Base64.toBase64(JSON.stringify(resale)) ]
    })
    console.log(result.data)
  }
}

main()