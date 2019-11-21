#include<ontiolib/ontio.hpp>
using std::string;
using std::vector;
using namespace ontio;

namespace ontio {
	struct test_conext {
		address admin;
		std::map<string, address> addrmap;
		ONTLIB_SERIALIZE( test_conext, (admin) (addrmap))
	};
};


class hello: public contract {
	public:
		using contract::contract;

		void mypanic(string &str, test_conext &tc) {
			if (check_witness(tc.admin)) {
				check(false, str);
			} else {
				check(false,"check_witness tc.admin failed");
			}
		}

		void mydebug(string &str) {
			printf("enter mydebug");
			printf("%s\n", str.c_str());
			printf("out mydebug");
		}
};

ONTIO_DISPATCH( hello, (mypanic)(mydebug))
