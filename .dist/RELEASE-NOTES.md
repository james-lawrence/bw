commit 1ac4e2829cc38bb4c32c6facf1dbf12daf8da27b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun May 17 07:24:57 2020 -0400

    add elbv1 and renamed elbv2 in interp

commit 75af1a2c4f529eeb31a52d49d7b3366983b2d30b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat May 9 12:09:14 2020 -0400

    add per command environment variables

commit a204fe0e2995b65d42239b5bb08afcd28c220e02
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 26 18:11:22 2020 -0400

    properly escape percentage

commit f79fa27ec223a73f0f0f3937aa7e09b4e9dbfc05
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 25 10:15:09 2020 -0400

    improvement debug notification when building locally

commit 496d079c1a12ce037cfbbe779316cf5aa899af83
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 24 19:34:48 2020 -0400

    add some color to user friendly errors

commit 4841c4cb178ace5d22cb7dfdd88ba8e618321823
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 24 19:30:47 2020 -0400

    add a version command.
    
    `bw version` will now display the version.

commit 44a3c4dc0a542d9a71f671eeb3c553c79eae2163
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 24 19:09:26 2020 -0400

    implement check to ensure an environment exists
    
    returns a useful error message to the user.

commit 833bc1d95a4cb831296311c23f014496f5ce1da0
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 19 17:01:30 2020 -0400

    add me show to display local user credentials

commit 69f62d8a101ed00ebed6b3417d271fd085e9ad49
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Apr 2 15:18:54 2020 -0400

    release 0.1.1585855648-35c2adbd

commit ee1318929cf806952aaefc15e51eff986a750d53
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Apr 2 15:17:19 2020 -0400

    implements reseting of the raft storage
    
    - fixes an issue where raft log errors would prevent promotion into
      quorum.
    - data isn't strictly necessary so its safe to do.
    
    also fixes a couple of tests.

commit 718a9d4c0730bb31e6fa6de6adfdc5ca056b68b9
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Mar 29 12:35:42 2020 -0400

    formatting cleanup

commit 1b5daa2095d34ec64b078d340cab65fa13e93583
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 21 13:35:26 2020 -0400

    release 0.1.1584811274-d4e8f3a3

commit d4e8f3a3b7a86f0f7d37dd4af022c810fe55f798
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 21 13:21:14 2020 -0400

    cleans up agentctl commands.
    
    adds a new agent command to print the state of the raft cluster from
    disk.

commit 78fe45d3d5f8bbaa04e4fb1fc201d4171b6b0cdb
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 21 12:15:26 2020 -0400

    improve raft code.
    
    - prevent panics by switching to file backed snapshots.
    - properly close the transport in failure cases.

commit 6e2f4bc1491026c2256a0c8617fba10b25772157
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Mar 15 10:48:31 2020 -0400

    release 0.1.1584283477-80cfa4f6

commit 80cfa4f6cddaaf7609717075eaf7079791340fe2
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Mar 15 10:44:37 2020 -0400

    fix test configuration

commit 75979a738377d69a286a652347942c36610f3e25
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Mar 15 10:34:01 2020 -0400

    test configuration improvements

commit 322d9bade1c16697f832c31e72aa04bc18d60b2e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Mar 15 10:33:26 2020 -0400

    prevent TLS request from spinning forever

commit e82fe56685569d32770d21e9f2deeeff95054f76
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 14 10:16:07 2020 -0400

    release 0.1.1584194492-72e186ba

commit 72e186baeb493bc5694328cd0810204147885787
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 14 10:01:32 2020 -0400

    more robust raft management

commit ec2fcb06dd35e4ee3f51d60876397cadf83bfb49
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 7 13:53:14 2020 -0500

    release 0.1.1583604757-d860d338

commit d860d338e9e1b6d81196bdd75a10a19a9770d21b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 7 13:12:37 2020 -0500

    release 0.1.1583604592-a5c568bd

commit a5c568bdf748d76766d45a6635c2907dd38e3ea6
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 7 13:09:52 2020 -0500

    increase peer checkin to 15minutes

commit bb8c5d747ec68dc2baabf123ee20b5bb0d433479
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 7 13:08:08 2020 -0500

    release 0.1.1583604375-3d98ef1f

commit 3d98ef1f83f8a6e008654804056ccd41278eda1b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 7 13:06:15 2020 -0500

    improvements to raft configuration
    
    1. stop using in memory store in non-test scenarios
    2. set a maximum of 1 minute between checkins
    3. allow leader to transfer leadership when its no longer a member of
       the cluster.
    4. only drop a single server from raft cluster per attempt.

commit be473192bae34f5eb9db3ebbccf38563e6091ec5
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 7 13:05:03 2020 -0500

    update dependencies

commit db22120becca48a43672640388ee7be048c868a8
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Feb 15 10:52:18 2020 -0500

    release 0.1.1581781289-4d9b077d

commit 4d9b077d506f670db2fc22378cf9be09dcebe693
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Feb 15 10:41:29 2020 -0500

    update torrent library - fixes deadlock in download

commit 2e16b9a5317d674e3385091b133c2fab7b1d6b4f
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Feb 15 08:06:39 2020 -0500

    remove dependency of anacrolix torrent lib

commit a1335ac772e1c61c0027355a313f8546fa8ed843
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Feb 15 07:03:53 2020 -0500

    bugfix - ensure release has version properly set

commit 9835821d018b5d6295fd878a8ed601d7be15ed6b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Feb 15 06:57:45 2020 -0500

    release 0.1.1581767305-393f9356

commit 393f935666e09278614a6d2ad6c6581f5a50dd79
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Feb 15 06:48:17 2020 -0500

    remove some extraneous logging

commit e09327e451310453f4dc735045b90ccd5571edc7
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Feb 14 23:39:14 2020 -0500

    release 0.1.1581741545-26aabbf8

commit 26aabbf863a8e54128865d4306c4651e91562dcb
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Feb 14 23:30:16 2020 -0500

    update to golang-1.13

commit 64a5a2e2b4424c56ed193ef05d8ccf7db5a36fbf
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Feb 8 13:58:56 2020 -0500

    update documentation

commit 345280c296967f20047bf9448c7f09294426f861
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Feb 7 19:08:13 2020 -0500

    create example client config

commit 3dc29043c276a07fc38fe7136bd9f3772bed1f3f
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Feb 7 18:53:45 2020 -0500

    improve logging information for directives

commit 55103c4f8900746649d418be1d9d71bf015b3d62
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Feb 6 18:39:17 2020 -0500

    minor fix to support canary

commit f8ec01dab8966141dd6b5a70686da025bfbe8807
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Feb 6 18:12:31 2020 -0500

    learn to prompt prior to deploy if specified

commit 5293660574ff563b3bf4a7fa566bbacf1c4562a0
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Feb 6 17:59:42 2020 -0500

    learn the canary command line option
    
    deploys to a consistent server without having to manually specify the
    filters.

commit e527abd0c5197c76f31f1aec1be43f2e5db623d0
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Feb 3 07:56:19 2020 -0500

    fix bootstrapping

commit 7c11dc09e22959c3685ac3d937f374426264f75b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Feb 2 14:04:08 2020 -0500

    update raft library

commit fa4352cedd65c6a25fa566b1d0d06f82bf6410a9
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Feb 2 13:51:41 2020 -0500

    update torrent library

commit 4703f63352172a5dbd56aca8987d2a06efd5cb3b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jan 26 18:07:47 2020 -0500

    fix darwin builds

commit e14e34872c28197cc37b43330f9d47e7b68ea13c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 18 08:21:01 2020 -0500

    cache private key used to generate certificate

commit 391e5a424c0a2b903a787523d8feeb94e248a105
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jan 12 11:17:05 2020 -0500

    cache the certificates generated by acme.
    
    unfortunately this isn't working atm because the private keys mismatch.

commit 4678f52340751f5933abb70520b46e63f8d547a0
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 18 06:52:40 2020 -0500

    release 0.1.1578845762-ff755cc0

commit ff755cc07cfdd0390b67c63da04edc1e87bcbd43
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jan 12 11:16:02 2020 -0500

    minor cleanups

commit 744dfc1abd330e113cec186c94157513e3a5f0f5
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 11 09:49:13 2020 -0500

    enable systemd interp module

commit 1ba4e4c6d7ea71953f5278c50a6ff1f5ee3b272f
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 11 08:56:18 2020 -0500

    fix invalid documentation

commit ce02b6c70cf8dd4abcd20c113a9e5b54ccec8868
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jan 10 18:08:27 2020 -0500

    implement golang interpreter
    
    - enables using golang to control the deployment.
    - simplifies some interactions where file or bash script are painful (such as detaching/attaching to a loadbalancer).

commit 1edc1cdc7347787557f3b8df6312188829e02ca2
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jan 10 18:08:53 2020 -0500

    update vendor

commit 7e5c07a68d328e2b98a66bff3fe7d273e7f938dc
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jan 8 12:25:50 2020 -0500

    release 0.1.1578504325-2484577b

commit 2484577be27924aa1eb0df2f6fd3353e5351628c
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jan 8 12:25:25 2020 -0500

    support gcloud instance managers

commit 62c32d58352e140c349093400e628a7d02ed55a9
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jan 7 16:18:18 2020 -0500

    fix it

commit 7faa495020092c32037e8792768758aa8f87b18c
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jan 7 15:02:01 2020 -0500

    release 0.1.1578424814-72e88d6c

commit 72e88d6c8f001d221a5978500f8378449685b0ce
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jan 7 14:20:14 2020 -0500

    add fallback to handle deprecated port

commit 9f116c3320d63915dc4cc401b5488f407c7288bb
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jan 7 14:15:13 2020 -0500

    release 0.1.1578424283-549f9ba2

commit 549f9ba29cd0f3c3ed4c4629f7ae7e5ecb3b3fa5
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jan 7 14:03:34 2020 -0500

    add environment variable for bootstrapping peers

commit 3f9755655ac09fac1ac8118107cf256c1faf06d3
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jan 6 09:11:03 2020 -0500

    merge filtered deploy into standard deploy

commit 3681fe14f9055ca2750587b51d67f2b429bc9ce9
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jan 6 08:56:45 2020 -0500

    release 0.1.1578318995-1ce533b5

commit 1ce533b54ec370eb6c28c3f9582f2cbc8d607d2d
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jan 6 08:56:35 2020 -0500

    cleanup documentation

commit f1281d9191b687bba4e88c26a58d04792d31d5ca
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jan 6 08:15:44 2020 -0500

    fully working proxy deployments

commit 4f8a4e7d0be27e32b4b02188046ab52153cb98b1
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jan 5 18:09:10 2020 -0500

    work todays fully proxied deployments

commit dd2beebea93ac85d654a013321d00de6640113fa
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jan 5 15:25:43 2020 -0500

    start the removal of the cluster from the client

commit 3756c40aa50e5acce61aa98e10f5d598de074576
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jan 5 14:35:40 2020 -0500

    start building proxy layer, with a better dialing interface

commit 39229e017982669dcbf45f5ce1b6cd36e7b3c2cd
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jan 5 12:27:47 2020 -0500

    fix up authorization setup

commit 6701f4037231271af88f0f31e3156b46f318c058
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jan 5 08:19:43 2020 -0500

    release 0.1.1578180264-dfdc8191

commit dfdc8191f548efd00748c3f27f41bab0812e1484
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 4 18:24:24 2020 -0500

    cleanup local development environment

commit 6608e210893993bf7f9d7ee4f542efcdd26811b5
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 4 18:21:41 2020 -0500

    tidy/vendor

commit fa1dc0448e6eb223d14e856b1b05789151e28e31
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 4 14:10:52 2020 -0500

    ensure notary works when discovery is public

commit 0a0909e326938255310c4fbf5721e32d9ae1eaa1
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 4 11:28:53 2020 -0500

    release 0.1.1578154854-ab6a0423

commit ab6a042361a66ea7b39d5da0289ba67a8c159ea8
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 4 11:20:54 2020 -0500

    update documentation

commit 8338695a79734859a940b38bb90cc57061417525
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jan 3 17:30:53 2020 -0500

    cleanup initialization of the agent

commit d932e440d285f6982f5d2bceea9565ac3bd1a6d6
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jan 3 17:29:41 2020 -0500

    update notary service to be more flexible

commit fcb9ef79ec6ad097a6b4fe8d833a81837e177f80
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jan 3 17:26:29 2020 -0500

    documentation updates

commit 462cd39bd29b6b0e1729677537b0a636909e56d1
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jan 2 21:09:58 2020 -0500

    make ssh authorizations configurable

commit 6fc2421c40fb521b935958d7b98a52415100df1f
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jan 2 20:16:18 2020 -0500

    support specifying network addresses from environment.

commit 1b68c91496fabbd159a5d8162d23f2ade1dd0fe6
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Dec 31 10:36:21 2019 -0500

    fix up client

commit 433ab832111cfc696fcafd26799387df59988f86
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Dec 31 08:25:18 2019 -0500

    release 0.1.1577564820-fee02853

commit fee028533d187e94bf4dddd8fb0723943ff1f472
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 28 15:27:00 2019 -0500

    minor example fixes

commit dd5c326be1416e4c85d9fe30e26d8d7d31c6d453
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 28 14:55:26 2019 -0500

    allow for DNS ACME challenge on google cloud

commit c86b03ea2207bfe1a30529cdd5b01e35ccc421a3
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 28 13:00:54 2019 -0500

    gcloud example

commit dc54a2e9c3675ce82158cf4764ef5ac71fc9ffd0
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Dec 25 13:03:08 2019 -0500

    release 0.1.1577296955-f5ba6783

commit f5ba67835091b59048fb83c232be7e84b64ebff3
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Dec 25 13:02:35 2019 -0500

    work towards dns challenge

commit 899543b70588527f161c31f0b625beead28c4470
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Dec 25 12:39:56 2019 -0500

    support gcloud load balancers

commit ff208fde491157c1ac77d883afa591aca5a3b813
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Dec 23 16:06:27 2019 -0500

    release 0.1.1577135182-8937bc73

commit 8937bc73cb6b7a5c8571b5af8f555a3cf2a10be4
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Dec 23 16:06:22 2019 -0500

    fix build

commit 90f8d49c1f56ead03ac285602781edb0c5402414
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 19 18:33:57 2019 -0500

    work towards alpn working in production

commit b5e572ff5c136d1e49f97e3e8275f3c8ebb8a956
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 19 06:55:41 2019 -0500

    release 0.1.1576756381-fa38f081

commit fa38f0817261c336c29a8454b86d44edcae2edbc
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 19 06:53:01 2019 -0500

    auto generate account key

commit d7ed78ea097e3dfde4e20cad25aac78b0119f427
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Dec 18 12:10:02 2019 -0500

    release 0.1.1576688989-a4210932

commit a421093220a30038bda86865725eec94a6cd36a1
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Dec 18 12:09:49 2019 -0500

    sync every hour

commit 896adbd6146e9ee693dda04e6cadddbdc655cd80
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Dec 18 02:18:45 2019 -0500

    working tls sync

commit ce969962680fef70fddc00f1d1a57c3b21c87c40
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Dec 18 00:21:38 2019 -0500

    work towards TLS propagation

commit 134dc7beded62712d561660dfe28b2f9c7be738b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 15 20:03:49 2019 -0500

    test fixes

commit 593728899195b5378c2b802d88a121931a4c5967
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 15 19:56:31 2019 -0500

    release 0.1.1576457713-e0fbf9ee

commit e0fbf9ee3207ce7f2ef96609214501421811c3bd
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 14 18:16:23 2019 -0500

    makes the WAL more extensible.
    
    - converts the WAL into a true binary log file. successive records
    written one after another.
    - maintains WAL backwards compatibility.
    - clients can now detect when their cacerts are invalid and properly
    dispose of them.

commit 69f0539ebb995e1a0b0eb4851dc8cca48fc5e7b5
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 15 10:09:57 2019 -0500

    fix state machine tests

commit d455c27bbfc535b2d0abc86898cc34c37e6957ed
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 14 12:37:41 2019 -0500

    working refresh endpoint

commit cc08a37f91b40dfead8add2998d6585dc6f2e29f
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 14 10:46:01 2019 -0500

    standard agent configuration loading

commit f8fdaa81d7b3676f4452bb10f29434d5cd1aee74
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 14 10:37:27 2019 -0500

    cleanup acme setup

commit ad4c2e09b9ab0896886a905e3e9b10152a6c6e0c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 8 16:30:22 2019 -0500

    state machine initialization

commit 4f0709f62661852616597e9d4eb7f441a99f1991
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Dec 10 08:59:14 2019 -0500

    release 0.1.1575825751-85ffe2d1

commit 85ffe2d1c6816a6d88c486a0bd56a860a58a2ca8
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 8 12:22:31 2019 -0500

    some test fixes, dependency updates

commit 13ef8d51abb56c97612107e1c1a1d7ba37e9cfe2
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 8 12:07:37 2019 -0500

    release 0.1.1575824833-74bafb83

commit 74bafb83471d59ee9ba46a7aac1031d61f975543
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 8 12:07:13 2019 -0500

    release 0.1.1575824660-ad5d15fe

commit ad5d15fe37755fb903ff1e25107deff95e7069c4
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 8 12:04:20 2019 -0500

    clean around notary service
    
    - most of notary service is now working, mainly refresh and search methods
    - more cleanup will be needed around ssh credentials. mainly ux related.

commit 44c3eadd7a833c94e59df82179f924e914257fef
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 8 09:30:12 2019 -0500

    misc cleanup towards discovery/notary endpoints

commit c22b184f4a419b57e1fc5c43a7b1ff298f999e48
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 7 16:31:27 2019 -0500

    first pass at agent configuration sync

commit 06c9d641a0f993b5da2095e1767ac8e7da85e466
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 7 13:10:21 2019 -0500

    add certificate cache authority directory

commit a2b65afb2e1468e1f4dfa4e9a71c88ed7d8cb189
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 7 12:26:07 2019 -0500

    working discovery service

commit 6f590764aa99766fe2820491da92b6d49caac880
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 7 10:57:24 2019 -0500

    almost working notary endpoint

commit 20c92cd300f6f5b0a92f4bad248b2205937582db
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 7 08:46:40 2019 -0500

    rework TLS setup to remove dependency cycle

commit 5c7559396779153fb4a6c5ba71776f2aa19726a4
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Nov 24 13:45:49 2019 -0500

    storage, basic auth setup, client, service tests, and persistance remaining

commit bc430435fa90ef824bef8755feb74898f515db0a
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 5 21:45:23 2019 -0500

    release 0.1.1575600321-74082f2c

commit 74082f2c4cf711d8e3051b1f4b039388c325fdab
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 5 21:45:21 2019 -0500

    build fix

commit cf7f27418f0c3948b952b70dcb0760cf47114028
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 5 21:27:34 2019 -0500

    give routine dump a fallback option
    
    if we are out of file descriptors we'll be unable to dump
    the routines due to the file failing to open.
    
    when this occurs fallback to stderr

commit 7cdc1f39b1b29adad2c131df04fc29399df398f2
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 5 21:03:02 2019 -0500

    cleanup connection each iteration

commit 037232b620b9ca36af2fffb183c281642f6a444c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 1 13:46:12 2019 -0500

    refactor agent daemon into daemons package

commit f45b218644a22edd1d6d81673edbfdd7b18c42cd
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 1 11:50:18 2019 -0500

    refactor bootstrap

commit 4f040637cdb805feae5f69b8a4400303997debf6
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Nov 26 11:23:10 2019 -0500

    minor fix for release code

commit 8ead3afa69391cc8b4c7d9c9b05fd76aed55165a
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Nov 26 10:04:01 2019 -0500

    support releasing to multiple ubuntu distributions

commit 047a5b37242ae0bd2fb1425dbb3c8cc7a0a6aa5c
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Nov 26 09:26:28 2019 -0500

    update distributions

commit 03556263a6ff1f31f8346460306508f459e8e8e1
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Nov 24 14:02:49 2019 -0500

    test fixes

commit af8a73638f27c63d6208f0ede1f1d38a94900f4d
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Nov 21 10:10:45 2019 -0500

    release 0.1.1574342862-7e5b93bd

commit 7e5b93bd8452b8e3922495a64de886d79fb31dd0
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Nov 21 08:27:42 2019 -0500

    release 0.1.1574177305-ff5c89c7

commit ff5c89c72608207882ccdf4d73189c8a689effa7
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Nov 19 10:28:25 2019 -0500

    make dead nodes slower to detect

commit 52828d3a8788741637901c97682cf9d5b84a0e40
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Nov 15 07:51:25 2019 -0500

    fix local deploys

commit 8789a31800a77f4c831de89e13c71facf7ca2e06
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Nov 13 12:14:32 2019 -0500

    another fix

commit b6ce43201a0459d08ca50d3d8c89707e9c1c3ffb
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Nov 13 12:11:38 2019 -0500

    fix archive root creation

commit 690fd0618d3f23fa5e567e83167e98ad9a8adc24
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Nov 12 16:11:44 2019 -0500

    release 0.1.1573593061-ea99721c

commit ea99721c20631d64181d75f0350f95a9805386a1
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Nov 12 16:11:01 2019 -0500

    release 0.1.1573593039-d7d9ff7c

commit d7d9ff7c59a8df5aa72db7c90b24a539b889f01e
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Nov 12 16:10:39 2019 -0500

    release 0.1.1573593036-d517f3ce

commit d517f3ce07291a169def59210f8c521ad08ac5bb
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Nov 12 16:10:36 2019 -0500

    fix reset

commit 3729ddaaf86ff818f40a1edee00197ce686bcdb5
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Nov 11 12:10:07 2019 -0500

    release 0.1.1573492097-aff3261c

commit aff3261c5dd39b797303a786263b8135ed659018
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Nov 11 12:08:17 2019 -0500

    release 0.1.1573490543-8ff38c00

commit 8ff38c004e7177c02cd7055ab9685870bb2327c4
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Nov 11 11:42:23 2019 -0500

    fix local

commit f6c0b94d0e0e2ce6044a38f59eec85395270d390
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun May 12 21:20:26 2019 -0400

    implement acme support.

commit 015b79eb8e9d0ced74ef9e173b9b8c1042ef2698
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Nov 9 04:53:30 2019 -0500

    release 0.1.1573293189-d9cb7395

commit d9cb7395ffa1a69d8c88c2e08a25ea45ae8c039d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Nov 9 04:53:09 2019 -0500

    release 0.1.1573293053-ad336746

commit ad336746fbf6bdaa1a6f7b54382cbf2716331e34
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Nov 9 04:50:53 2019 -0500

    fix redeploy to properly wait for completion

commit 449ac4a9a48c8877e5c5a611d095fedba4bd9dd8
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Nov 9 03:52:00 2019 -0500

    prep deploy directory when a deploy context is created

commit 2c505ab08f7098996102607d51e72de406cdd29a
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Nov 9 03:07:47 2019 -0500

    implement redeployment of a already uploaded archive

commit ec07243a80394baf9e23e904897998ac82210ffe
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Oct 29 21:02:44 2019 -0400

    release 0.1.1572397254-4893a7f9

commit 4893a7f9711520ffe13a5ab224de4f1923b05101
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Oct 29 21:00:54 2019 -0400

    readd bootstrap

commit 1298e2a39f95c34e8dca426987f90fd1f2198e5a
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Oct 29 20:37:11 2019 -0400

    fix info timeout

commit b9843fac454b33e2257724c806bc773b10e07e45
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Oct 20 19:46:04 2019 -0400

    release 0.1.1571615162-48ec32ac

commit 48ec32acae0dd5db1661b4856ac052247686dbda
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Oct 20 19:46:02 2019 -0400

    fix vendor directory

commit 507b2ec81e8181bcc37e3df4305a0be95c58d038
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Oct 20 12:57:27 2019 -0400

    release 0.1.1571590523-059830fb

commit 059830fb308acf565a1f173b7e809297e0b04d0d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Oct 20 12:55:23 2019 -0400

    release 0.1.1571590518-2812c76b

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
