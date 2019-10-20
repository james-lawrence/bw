commit 2812c76b1399b2acfa17a337f280ccf1be97b619
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Oct 20 12:51:37 2019 -0400

    release 0.1.1571590287-6569ef6c

commit 6569ef6c2263c0223b0e85894965fa58f79a4946
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Oct 20 12:51:27 2019 -0400

    revert to master

commit ff23155c4e2c2b106302101d23c7d52b74425bdc
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Oct 20 12:49:53 2019 -0400

    release 0.1.1571590168-9ba27031

commit 9ba2703130a0c3a067d437cd8599378eaa7d8750
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Oct 20 10:29:57 2019 -0400

    debugging

commit fae8f1f85369d44c55fc7459a434361827170f64
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Oct 20 09:29:51 2019 -0400

    fix api changes

commit 8c46767d53b877d1cd58b4de30dbefdfb73e46b3
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Oct 20 08:48:48 2019 -0400

    update torrent library to fix deadlock

commit 6ec9c8b07f67a8dd8fbb42e6464cde0b2676cc27
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Sep 19 16:41:13 2019 -0400

    propagate config environment to shell directive for local deploys

commit 6f01e242570be2a2b2fe427311cf44a94826a2e0
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Sep 15 11:45:01 2019 -0400

    release 0.1.1568562279-421603f7

commit 421603f7c3ad52bf0d5baa10e4cdce208a26d084
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Sep 15 11:44:39 2019 -0400

    fixes for filesystem bootstrap

commit 0f32f465c4a295cf0ca939904086f6da18e47819
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 14 20:28:48 2019 -0400

    release 0.1.1568505946-104682d8

commit 104682d8bf4f0f9b3d0d269b598b089459a9bba4
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 14 20:05:46 2019 -0400

    release 0.1.1568505016-48229414

commit 48229414ffe39ffdb28f5752a740f721e7d5c5e2
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 14 19:50:16 2019 -0400

    release 0.1.1568504947-293e88a1

commit 293e88a11308ad5f32a9dfc33c8a955fa4504622
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 14 19:49:07 2019 -0400

    final touches to filesystem bootstrap service
    
    - upload deploy now is consistent in its deployment ID, identical
    archive = identical deployment id.

commit df84130750e46e8123f4028fa80729f61e7da940
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 14 18:48:42 2019 -0400

    working filesystem bootstrap

commit 7fd9e15681e8f99664cc42c672b3710f7798d4ac
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 14 14:45:00 2019 -0400

    release 0.1.1568486697-0de1176a

commit 0de1176a47a156067452ba40addce2220317b87e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 14 14:44:57 2019 -0400

    fix build

commit 9844e7c5e190497fa7452df9b94f07aa12f76f0c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 14 13:26:31 2019 -0400

    release 0.1.1568481965-9aa2c457

commit 9aa2c4574f87e80b688159eaaa725c96461efbcb
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 14 13:26:05 2019 -0400

    release 0.1.1568481892-f2d3c721

commit f2d3c721f34109684833a64c9257913d09c43b47
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 14 11:07:36 2019 -0400

    fix up dependencies

commit a3fb699db6da553229e0728c0d74b9a7d67837b7
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Sep 13 20:19:33 2019 -0400

    initial bootstrap services refactor

commit 47600a80e5294bbb3f505dba616aae3a6e3c1b54
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jul 4 19:50:44 2019 -0400

    improve shutdown directive to cleanup bad deploys

commit 98b0f7b4c479ef0616d3030797c512a15058bc84
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jul 4 09:53:00 2019 -0400

    improve restart reliability

commit 0e7211bac37d966f84ff46d4af22d2c6eb3840de
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jul 4 08:05:21 2019 -0400

    do not use the state machine as the dispatcher
    
    if leadership is changed during a deploy proxy service
    won't properly dispatch messages.

commit f3e8672968d1455338be29d52286010b8aa436e4
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 26 20:30:18 2019 -0400

    update dist debian build rules

commit 312ca5307ab01e906ca77a4c151f03ac01c74300
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 26 20:23:11 2019 -0400

    release 0.1.1561594732-9980085f

commit 9980085fcde86961ac986994d3913cab7f7c915a
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 26 20:18:52 2019 -0400

    release 0.1.1561594711-b6dfb78a

commit b6dfb78a017dc7ee9906593ad5d44aa485e351f4
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 26 20:18:31 2019 -0400

    release 0.1.1561594672-c9b03977

commit c9b03977035e96fe7b200a1752cdc57562836f84
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 26 20:17:52 2019 -0400

    release 0.1.1561594352-28faea42

commit 28faea42ba5ec55ae8cb721df9ebec14e6ebca34
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jun 24 18:57:55 2019 -0400

    release 0.1.1561416854-4464e66b

commit cdad914d7d23ad554a2ddade4fbe983b3b4357b4
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jun 24 20:16:38 2019 -0400

    include dispatcher type when ReliableDispatch fails

commit a6ef521efc3a8b681b464b153a11f06f5e654e54
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jun 24 18:51:19 2019 -0400

    release 0.1.1561416315-44980f3a

commit 44980f3a0553f864b9b5799701d8b3ca1113f588
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jun 24 18:45:15 2019 -0400

    fix archive directory

commit 9c29dcaf64207297268a824b4f72e6e2d49bf90b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 16 22:27:44 2019 -0400

    release 0.1.1560738341-955e99c5

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
