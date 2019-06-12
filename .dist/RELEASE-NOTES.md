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

commit 5b153204b9c95fa7889ecc7faa36b99878e401f8
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 5 17:59:57 2019 -0400

    release 0.1.1554501589-dbfc9357

commit dbfc9357cc97ceb57d49aea526a33dda6c5138d9
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 5 17:49:40 2019 -0400

    update build process

commit 5dab47ccd5e451c4665f0d31f6327bd9342e76ef
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 5 17:35:06 2019 -0400

    release

commit 2669f836e63efdae37a690cd5ca449fc643ce8f2
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 5 17:23:47 2019 -0400

    improve notifications

commit dd18ba24beb1f567e27868b756e9d20fd5eb1efb
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Mar 27 21:56:43 2019 -0400

    Improve ux - handle deployment restarts
    
    - update cli client to handle restarts

commit 02ae0c10a6793e29bf45d01726462dc12934b3c0
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Mar 27 20:33:02 2019 -0400

    add a restart event to deploy commands

commit 4e9a7fdd2d136e56ce725cc301cc98e422b017dc
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Mar 27 19:51:30 2019 -0400

    misc improvements to client
    
    - fixes file handle leak when connection to cluster fails
    - remove some extraneous debug logging
    - silence some noisy logs by default

commit 7524bdcd1bae373203188c405efc9f603ff6acc5
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 23 14:16:40 2019 -0400

    move extension packages into internal package
