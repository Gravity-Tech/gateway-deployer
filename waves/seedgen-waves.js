const { seedUtils } = require('@waves/waves-transactions');
const { Seed } = seedUtils;

function generateSeeds(chain_id) {
	const keys = 'Distributor,Gravity,Nebula,Port'.split(',')
		.reduce((acc, key) => {
			const chainId = chain_id.charCodeAt(0)
			const instance = new Seed(Seed.create().phrase, chainId);
			let { address, phrase: seed, keyPair } = instance;
			const { publicKey, privateKey } = keyPair

			return {
				...acc,
				[key]: { address, seed, publicKey, privateKey } 
			}
		}, {})
	
	return {
		...keys,
		Seeds: {
			"DistributionSeed": keys.Distributor.seed,
			"GravityContractSeed": keys.Gravity.seed,
  			"NebulaContractSeed": keys.Nebula.seed,
  			"SubscriberContractSeed": keys.Port.seed,
		}
	}
}

const grepParam = (paramName) => {
	const i = process.argv.indexOf('--' + paramName)
	return i !== -1 && process.argv[i + 1]
}

function main() {
	const chain_id = grepParam('chain') || 'S'
	const seeds = generateSeeds(chain_id)

	console.log({ seeds })
}

main()

