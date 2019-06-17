commit 955e99c5a5f0d55dcb9220a06b016bb47e085dc5
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 16 22:25:41 2019 -0400

    release 0.1.1560738334-6b988d1b

commit 6b988d1b1f4c6ae53439af90fe53c15e38e3a062
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 16 22:25:34 2019 -0400

    disable generate during release

commit 2e42c090c633cb07dc62d7852a889725b05ef3fa
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 16 22:23:37 2019 -0400

    Revert "package update"
    
    This reverts commit 1bcf16ef4bd0172f877c046e71a637c77b8465cc.

commit 1bcf16ef4bd0172f877c046e71a637c77b8465cc
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 16 21:33:56 2019 -0400

    package update

commit 444b1893c6a622df0cc99ad4eae701a6aad71d09
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 16 22:06:35 2019 -0400

    fix build

commit c8f43c72a6481f7a86b0381256ba6d5290e15555
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 16 20:04:05 2019 -0400

    release 0.1.1560729812-a3051622

commit a3051622edac34d908cda19ea58f2e8250a3a66b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 16 20:03:32 2019 -0400

    release 0.1.1560724222-67063fd1

commit 67063fd131b12d736c4cf4d42abab2c39810ee3a
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 16 18:30:22 2019 -0400

    fix deploy initiator key

commit 4c894bd725e03c974e8e317527e7faa573988449
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 16 18:11:38 2019 -0400

    minor setup/testing changes

commit eb8e99cf093a5d5d67d657d5a3c32395436b329d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 16 16:05:09 2019 -0400

    remove comment

commit 001a2bba8413a056ed6c98f33cd7bcc4a6b5a604
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 12 00:32:43 2019 -0400

    release 0.1.1560313958-439c2464

commit 439c24642b87ecec17bebec65da7d5e501a68c68
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 12 00:31:44 2019 -0400

    release 0.1.1560313898-35e29b46

commit 35e29b46a3de1b710ca5a3ad848e08a510dd0ec9
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 12 00:31:38 2019 -0400

    docker fixes

commit a532ed910f61edceb57d964b39d14396721b0426
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 12 00:22:25 2019 -0400

    release 0.1.1560313317-7af44a15

commit 7af44a15f9b1f9a41c592a6ca1eb9891add87b57
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 12 00:21:57 2019 -0400

    release 0.1.1560311662-96654a53

commit 96654a530c288b82f6c513a64fdf5f15b817bdd2
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jun 11 23:54:22 2019 -0400

    release 0.1.1560311658-a5e7877c

commit a5e7877c4f79f0bb69811ea2305422fc1551f4d0
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jun 11 23:54:18 2019 -0400

    fix makefile

commit 81daabb8081b78246127c13af7c6507547c5c9f1
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jun 11 23:53:03 2019 -0400

    release 0.1.1560311573-d78371d0

commit d78371d01f39ceb111fc0eba1fd8d35f3faf303a
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jun 11 23:52:53 2019 -0400

    fix up packagekit

commit afb7eddfc3a2d7dedf85ea51470f590ac2b268b7
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jun 11 23:44:25 2019 -0400

    add debug logging

commit a5cbf539097f61f5bc8997066aed805965ea8aee
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jun 11 23:37:38 2019 -0400

    fix minor bugs
    
    - special case macosx FQDN resolution, because macs do not setup their
    hostname network properly.
    - handle autocorrecting dead deploys when querying dpeloyments.

commit 6441213010756bd7444054f2600ea0315349f4a3
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun May 12 21:03:37 2019 -0400

    working CSR for acme configuration.
    
    work left to do:
    - ensure acme certificates result in an operational server.
    - ensure only clients from the same cluster can connect.

commit a239cf36d9c968c9f7e373c473b7389fd2aa1549
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun May 12 17:18:11 2019 -0400

    updates to acme support
    
    - support expanding newlines in environments for use in YAML file.
    - add documentation to certificatecache directory.
    - allow for acme to refresh its certificates.

commit dde816e8d0469ba31bff2d4438b67ce2f7babee4
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun May 12 10:08:40 2019 -0400

    work on implementing acme protocol support.

commit 51b8e5a29b22a81d329b1a95f0a761280dbd366b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 13 14:08:44 2019 -0400

    rename commands package

commit 4668603a3fc4772a529f59ee6e824bad88626681
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 6 07:25:24 2019 -0400

    minor cleanups
