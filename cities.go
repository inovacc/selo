package selo

// citiesByUF maps each UF to a small list of real municipalities (top ~10 by
// population). Used by GeneratePerson to attach a UF-consistent City to the
// synthetic Address. Source: IBGE municipality population estimates; vendored,
// no external deps. Synthetic-identity use only. Every UF MUST have >=1 entry
// (genAddressForUFRand indexes into the slice via r.IntN(len(...))).
var citiesByUF = map[UF][]string{
	UFAC: {"Rio Branco", "Cruzeiro do Sul", "Sena Madureira",
		"Tarauacá", "Feijó", "Brasiléia", "Plácido de Castro"},
	UFAL: {"Maceió", "Arapiraca", "Palmeira dos Índios", "Rio Largo",
		"Penedo", "União dos Palmares", "São Miguel dos Campos", "Coruripe",
		"Delmiro Gouveia", "Campo Alegre"},
	UFAP: {"Macapá", "Santana", "Laranjal do Jari", "Oiapoque",
		"Mazagão", "Porto Grande", "Tartarugalzinho"},
	UFAM: {"Manaus", "Parintins", "Itacoatiara", "Manacapuru",
		"Coari", "Tabatinga", "Maués", "Tefé",
		"Iranduba", "Humaitá"},
	UFBA: {"Salvador", "Feira de Santana", "Vitória da Conquista", "Camaçari",
		"Itabuna", "Juazeiro", "Ilhéus", "Lauro de Freitas",
		"Jequié", "Teixeira de Freitas"},
	UFCE: {"Fortaleza", "Caucaia", "Juazeiro do Norte", "Maracanaú",
		"Sobral", "Crato", "Itapipoca", "Maranguape",
		"Iguatu", "Quixadá"},
	UFDF: {"Brasília", "Ceilândia", "Taguatinga", "Samambaia",
		"Planaltina", "Águas Claras", "Recanto das Emas", "Gama"},
	UFES: {"Vila Velha", "Serra", "Cariacica", "Vitória",
		"Cachoeiro de Itapemirim", "Linhares", "São Mateus", "Colatina",
		"Guarapari", "Aracruz"},
	UFGO: {"Goiânia", "Aparecida de Goiânia", "Anápolis", "Rio Verde",
		"Luziânia", "Águas Lindas de Goiás", "Valparaíso de Goiás", "Trindade",
		"Formosa", "Novo Gama"},
	UFMA: {"São Luís", "Imperatriz", "São José de Ribamar", "Timon",
		"Caxias", "Codó", "Paço do Lumiar", "Açailândia",
		"Bacabal", "Balsas"},
	UFMT: {"Cuiabá", "Várzea Grande", "Rondonópolis", "Sinop",
		"Tangará da Serra", "Cáceres", "Sorriso", "Lucas do Rio Verde",
		"Primavera do Leste", "Barra do Garças"},
	UFMS: {"Campo Grande", "Dourados", "Três Lagoas", "Corumbá",
		"Ponta Porã", "Naviraí", "Nova Andradina", "Aquidauana",
		"Sidrolândia", "Paranaíba"},
	UFMG: {"Belo Horizonte", "Uberlândia", "Contagem", "Juiz de Fora",
		"Betim", "Montes Claros", "Ribeirão das Neves", "Uberaba",
		"Governador Valadares", "Ipatinga"},
	UFPA: {"Belém", "Ananindeua", "Santarém", "Marabá",
		"Parauapebas", "Castanhal", "Abaetetuba", "Cametá",
		"Marituba", "Bragança"},
	UFPB: {"João Pessoa", "Campina Grande", "Santa Rita", "Patos",
		"Bayeux", "Sousa", "Cajazeiras", "Cabedelo",
		"Guarabira", "Sapé"},
	UFPR: {"Curitiba", "Londrina", "Maringá", "Ponta Grossa",
		"Cascavel", "São José dos Pinhais", "Foz do Iguaçu", "Colombo",
		"Guarapuava", "Paranaguá"},
	UFPE: {"Recife", "Jaboatão dos Guararapes", "Olinda", "Caruaru",
		"Petrolina", "Paulista", "Cabo de Santo Agostinho", "Camaragibe",
		"Garanhuns", "Vitória de Santo Antão"},
	UFPI: {"Teresina", "Parnaíba", "Picos", "Piripiri",
		"Floriano", "Campo Maior", "Barras", "União",
		"Altos", "José de Freitas"},
	UFRJ: {"Rio de Janeiro", "São Gonçalo", "Duque de Caxias", "Nova Iguaçu",
		"Niterói", "Belford Roxo", "Campos dos Goytacazes", "São João de Meriti",
		"Petrópolis", "Volta Redonda"},
	UFRN: {"Natal", "Mossoró", "Parnamirim", "São Gonçalo do Amarante",
		"Macaíba", "Ceará-Mirim", "Caicó", "Açu",
		"Currais Novos", "Apodi"},
	UFRS: {"Porto Alegre", "Caxias do Sul", "Canoas", "Pelotas",
		"Santa Maria", "Gravataí", "Viamão", "Novo Hamburgo",
		"São Leopoldo", "Rio Grande"},
	UFRO: {"Porto Velho", "Ji-Paraná", "Ariquemes", "Vilhena",
		"Cacoal", "Rolim de Moura", "Jaru", "Guajará-Mirim",
		"Pimenta Bueno", "Ouro Preto do Oeste"},
	UFRR: {"Boa Vista", "Rorainópolis", "Caracaraí", "Mucajaí",
		"Alto Alegre", "Pacaraima", "Cantá"},
	UFSC: {"Joinville", "Florianópolis", "Blumenau", "São José",
		"Chapecó", "Itajaí", "Criciúma", "Jaraguá do Sul",
		"Lages", "Palhoça"},
	UFSP: {"São Paulo", "Guarulhos", "Campinas", "São Bernardo do Campo",
		"Santo André", "Osasco", "Ribeirão Preto", "Sorocaba",
		"Santos", "São José dos Campos"},
	UFSE: {"Aracaju", "Nossa Senhora do Socorro", "Lagarto", "Itabaiana",
		"São Cristóvão", "Estância", "Tobias Barreto", "Itabaianinha",
		"Simão Dias", "Nossa Senhora da Glória"},
	UFTO: {"Palmas", "Araguaína", "Gurupi", "Porto Nacional",
		"Paraíso do Tocantins", "Colinas do Tocantins", "Guaraí", "Tocantinópolis",
		"Dianópolis", "Miracema do Tocantins"},
}

// weightedToken pairs a string value with a relative selection weight; used by
// pickWeighted to bias address-token sampling.
type weightedToken struct {
	value  string
	weight int
}

// logradouroTypes are weighted street-type prefixes (Rua/Avenida dominate).
var logradouroTypes = []weightedToken{
	{"Rua", 50}, {"Avenida", 25}, {"Travessa", 8}, {"Alameda", 6},
	{"Praça", 5}, {"Estrada", 4}, {"Rodovia", 2},
}

// neighborhoodTokens are generic, nationally-plausible bairro names (Centro
// dominates). Deliberately UF-agnostic to keep the dataset tiny.
var neighborhoodTokens = []weightedToken{
	{"Centro", 20}, {"Jardim América", 8}, {"Vila Nova", 8}, {"Boa Vista", 7},
	{"São José", 7}, {"Santo Antônio", 6}, {"Bela Vista", 6}, {"Industrial", 5},
	{"Cidade Nova", 5}, {"Jardim Primavera", 5}, {"Vila Mariana", 5}, {"Bom Retiro", 4},
}
