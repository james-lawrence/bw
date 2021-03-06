commit a08ddbce4e74ad365e2a264c1d1505665df47d5b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jul 18 12:58:58 2021 -0400

    adjust shell lgoging output

commit 8fc1c38e35ff75f8d503df3d0c898ddc2007ad61
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jul 15 08:38:25 2021 -0400

    test environment fixes

commit a667add372c3a41a1a3a1b936af9e6e71cde66f0
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jul 15 08:16:23 2021 -0400

    possible missed path on cleanup of raft transport

commit b89ea74b927586511a289bed3530fc586cd8cf51
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jul 15 08:12:31 2021 -0400

    hack fix for the already bound protocol error

commit 301de2976598f305dafaf36c8d7d06036a687683
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jul 14 09:56:35 2021 -0400

    add deployment id to failure message

commit c3eb30e826208a73b27cfe03aa5051e91337dff7
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jul 13 10:31:27 2021 -0400

    add insecre flag to info

commit 9339a3c1449e26340ee4c7511f8bb655fc1a2a89
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jul 12 12:04:02 2021 -0400

    update insecure flag for cancel and redeploy

commit 3d91f3faba3c1cb7a5362df2cce9f1ec31fe4735
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jul 10 15:55:11 2021 -0400

    release 0.1.1625946372-0ed2c291

commit 0ed2c291c42c2baa9c3062b42c6592b08f2bb094
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jul 10 15:36:22 2021 -0400

    fix go.mod

commit 0e70a949bfd082fde84186cd36fc4ffebf6b4451
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jul 10 15:35:36 2021 -0400

    release 0.1.1625945648-7f130cd1

commit 7f130cd1cc280ad7e8e7d134d5e7fdf99e06d2fe
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jul 10 15:34:08 2021 -0400

    add feature flag for proxy dialing

commit e1e072f918e73ae261b021dde95fd9f5df60fd8f
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jul 10 15:25:50 2021 -0400

    fixes for proxy connections

commit 95dc09545da1ac6a45d86273eccc1fb1c441fab3
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jul 10 14:14:28 2021 -0400

    add lego tlscert

commit 323084725329ae872ddeccc87a45ce3590b71a02
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jul 10 13:01:35 2021 -0400

    update go.mod

commit 81ef8e2daa8ff38c9e2da63ab469607cabeebe7c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jul 10 13:01:28 2021 -0400

    cloud update

commit a82e1848a321c1a7ef7f166ffe0344033992e8e1
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jul 10 12:41:34 2021 -0400

    debug join

commit 9976814bdb097fcd571b7aa8ebde5935404e4070
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jul 10 09:45:43 2021 -0400

    update gcloud example to unlock tls acme protocol

commit 524f55d649010f9a7ce0f0ab0545079df4b18350
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jul 9 13:53:36 2021 -0400

    tls cleanup

commit 801783a5857da86a398f8c6f084291efc97699ab
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jul 9 14:02:05 2021 -0400

    revert logging

commit 3a438cb91dcc6c727d55dd317d2e8a753ffee1cd
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jul 9 13:59:21 2021 -0400

    debug mode

commit c1a82d1ec438589c6fdf4c0a62ad06ad5b2c663a
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jul 9 13:46:01 2021 -0400

    allow disabling the certificate refresh system.

commit 4844532d05d4df7ae39f45ed4da5101b75b9f4ec
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jul 9 11:23:23 2021 -0400

    update workspace generation using go:embed

commit 6b3ecdd72a5e3e3c11a6ec43c9eff202c3033df2
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jul 8 09:58:09 2021 -0400

    release 0.1.1625752276-eec9202b

commit eec9202b49dc566a4b58a7d7b7c9d2944e827840
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jul 8 09:51:16 2021 -0400

    fix deployment logic during WAL restoration

commit 5a1cbee6a894cf5283483d1ac227ab868753d5f7
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jul 8 07:53:53 2021 -0400

    fix overflow bug in exponential backoff.

commit 3cc5c536a6567d5f5d073d688284164d0b4af665
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jun 28 14:56:01 2021 -0400

    release 0.1.1624888692-dd9ddb0b

commit dd9ddb0b3a6b1db323a804902a226a9121934a72
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jun 28 09:58:12 2021 -0400

    always revert WAL to healthy, regardless of errors during restore

commit 9caa76821b80a43700203bf49a96138f8b4afdf7
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jun 25 18:43:21 2021 -0400

    release 0.1.1624659786-f3ca8ab5

commit f3ca8ab567a7008905eab31beaea4d3a017e3846
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jun 25 18:23:06 2021 -0400

    fix deploy notifications

commit adb6285861d54ec626f3884f459c703e45209ff0
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jun 19 08:34:09 2021 -0400

    release 0.1.1624104693-ad6bb6c7

commit ad6bb6c765a76425b4c8c888d638915aad24501f
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jun 19 08:11:33 2021 -0400

    release 0.1.1624103458-9480f9ce

commit 9480f9cec213b5b8600ed8ce3714fd5c5795be2a
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jun 19 07:50:58 2021 -0400

    code cleanup

commit 359f2151b9a82f2a35931e0987fac6d9b6adcde3
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jun 18 10:29:54 2021 -0400

    cleanup dependencies

commit 7f1fe58484d0690c44e1ec4d41bd68919abbb886
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jun 18 10:13:06 2021 -0400

    update hashcirp dependency

commit cdafa9e8a525f0a6b5f07d2b29084d20646d348b
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jun 17 20:26:46 2021 -0400

    release 0.1.1623975826-092cce0a

commit 092cce0a7dbfbeebaba1c883e8108d57770d1e5f
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jun 17 20:23:46 2021 -0400

    reduce sleep between checks significantly.
    
    speeds up deploys.

commit defccd327c9e8d395165dc71887d0b8da984b8ef
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jun 17 20:11:27 2021 -0400

    linting cleanup

commit fa6a410ebdcd010df62f9c119aa7a337669e707a
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 16 18:41:25 2021 -0400

    refactor event watch code

commit 5897deb3a01bd1ffd0865115c4496d3031f12ec6
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 16 22:07:44 2021 -0400

    release 0.1.1623895491-9f3f170f

commit 9f3f170f92c7185c9690b45b6526a88318ea7451
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 16 22:04:51 2021 -0400

    dependency fixes

commit 2f305f8291a34e0def741a0efd9286fe83bcc45a
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 16 21:59:50 2021 -0400

    dependecy update

commit 3fe0ac826f264ab984f62344b409d3da690c17e6
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 16 21:35:57 2021 -0400

    disable banning ips

commit b5133be52e613f8b8494312ee68568a67c21553c
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 16 19:04:10 2021 -0400

    update dependencies

commit 993106bd9af1c8cf4a783cc4f3bc4e357c5e45c3
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 16 18:00:40 2021 -0400

    release 0.1.1623880262-067e2c7f

commit 067e2c7f27422a2533c21115c6f5f3ebf840dff6
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Jun 16 17:51:02 2021 -0400

    fix build problems

commit a6bbec8335e55f7a03b12670e97d90725b938d7e
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jun 15 12:03:42 2021 -0400

    fix bolt import

commit d5272c12e06d12fa4aa89d54e6e9d83113c3b3b8
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 13 19:52:04 2021 -0400

    logging cleanups

commit 63ba10ddb902ceade4ec13ccff61d0dd20136d5c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 13 19:32:34 2021 -0400

    implement node proxy protocol for the socket.
    
    this lets use any node in the cluster as a jump host to any other node.
    
    requires deployment permissions.

commit 105b091c87585e6ed8a1730ee754a4c59375f398
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jun 8 11:49:44 2021 -0400

    wip

commit 5a8102171c44ba5b082d6235534c5360742d3d32
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 6 16:42:23 2021 -0400

    implement a tunneling protocol
    
    allows tunneling through a specific node through the
    discovery address.
    
    this allows for accessing nodes when their IPs are inside a private network
    accessible only through a proxy server.

commit 0250a4bd90fa7518a590c6b0142bd573a2f9823c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 13 09:35:23 2021 -0400

    release 0.1.1623590957-60d4255f

commit 60d4255f985d056a10e60d8b15b2c60ae7c592f4
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 13 09:29:17 2021 -0400

    debug logging

commit bbd84ae7b5f10ca5c998159f8c438815051e90e4
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 13 09:18:29 2021 -0400

    remove unsafe dialer for certificate generation.
    
    use key stretching to generate a deterministic private key from the
    primary token this allows bootstrapping securely.

commit f71e8d1d4c3c14a05abbff47dd3a456a3c22d8f9
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 13 08:54:53 2021 -0400

    generate deterministic bootstrap keys

commit 2fb34459975cbd74db666d3c12a319a242a57d99
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 13 08:54:07 2021 -0400

    cleanup dead code

commit 5932778286fcb6f95fc9433231924b4a366faf4d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 13 08:53:37 2021 -0400

    fix filesystem bootstrap failures

commit e6b7429cb66a881ffe6d7803a7bc2ca4e4c662f6
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 13 08:28:22 2021 -0400

    minor changes to how watch is implemented.
    
    1. add a 1 second timeout to dispatch for any given watcher.
    2. hard stop the watcher service instead of graceful, suspicion is that
    the server isn't being shutdown because it is streaming, but the client
    isn't receiving messages because we've exited the loop.

commit 684546d795ade08562975b007a325e1844292e4e
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jun 10 11:41:42 2021 -0400

    expose useful environment variable parsing functions to interpreter

commit 7f2c0ce0b562e1f5d33acb0274db7d1c5e63c007
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jun 3 19:11:59 2021 -0400

    cleanup

commit 1560f5b73f606a80f2b96050d0f234efd4988cb1
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Jun 3 19:04:13 2021 -0400

    client cleanup

commit 0aba9a5cbbc8ec14022e9523c9f2e72917bc1195
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat May 29 12:45:43 2021 -0400

    more cli cleanups

commit 4e1c4f0051fb3d1e9875b3d15e6c87d225da631f
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat May 29 12:07:11 2021 -0400

    terminal ux improvements

commit 58920941a8b73493842a9cfa37f5111145ac8393
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri May 28 13:39:21 2021 -0400

    release 0.1.1622223501-7277ae96

commit 7277ae962eb9e161fa9fb42830849205bcebd5e0
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri May 28 13:38:21 2021 -0400

    fix keygen

commit 708d091b3464b86c0632689d8f42a88b102b4881
Merge: 7dbd0a0a 8f49ab8e
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri May 28 11:07:34 2021 -0400

    Merge branch 'notary/agent-key-gen'

commit 8f49ab8e3ca30b87a6d34e3b0f100209aec0d1bf
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri May 28 11:06:44 2021 -0400

    fix non-determistic rsa key gen

commit 7dbd0a0a4f650af16ed51fe57921fbee23b7685e
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue May 25 14:51:37 2021 -0400

    release 0.1.1621968591-622a866f

commit 622a866f3864ad9d7f90e32369d953287989fc16
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue May 25 14:49:51 2021 -0400

    revert to working version

commit 648c636c27873fc998470aa6d83cbff8d92a123d
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue May 25 11:47:18 2021 -0400

    release 0.1.1621450873-95b7cedd

commit 51f00a4e737b73180ac333758a26602e64acefd0
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue May 25 11:39:16 2021 -0400

    debugging

commit 7a1f1abe1d102c100c1605e0b4b2e824eab6960a
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat May 22 12:03:07 2021 -0400

    release 0.1.1621699331-69f50639

commit 69f506390d853b62fd348f1a1dc5a3fcf3164ac7
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat May 22 12:02:11 2021 -0400

    remove detecting public keys from node events

commit 350f5592ce00ca8bb6b69389f866865498004ef0
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat May 22 11:57:10 2021 -0400

    release 0.1.1621698926-c4ef809f

commit c4ef809fd680340579ee4851c96ca8b44384e41b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat May 22 11:05:49 2021 -0400

    deterministic ssh key for agents

commit 95b7ceddb204827f61eab8208e3f77b8f4ed5cd8
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed May 19 15:01:13 2021 -0400

    release 0.1.1621450816-fb3dbe30

commit fb3dbe304edf4aa4078bc4c497cd7ab3cbbd44aa
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed May 19 14:58:20 2021 -0400

    attempt to properly handle rate limits

commit 85d63cf2ab4e38daae8277ecd3bb0bc7a1ba2228
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue May 18 07:21:10 2021 -0400

    release 0.1.1620999427-32c479fa

commit 32c479fa5ee6a035d64da63018b685315c7a4461
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri May 14 09:37:07 2021 -0400

    logging adjutment

commit 656996671d8c3e062f9105643a987f2f0d153354
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri May 14 09:25:36 2021 -0400

    release 0.1.1620998660-5417796c

commit 5417796c837ddf1715bdf6558f909414cf4b9311
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri May 14 09:24:20 2021 -0400

    implement retries for aws loadbalancers

commit 1d629e42b2e15bc1a84343bd822c5f003efd33f3
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed May 12 16:19:07 2021 -0400

    release 0.1.1620850675-f36df42e

commit f36df42e1bc7abdae0d5d74107dbed1ae8e96147
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed May 12 16:17:55 2021 -0400

    properly return an error on grpc request

commit b6a9551aed77b5566efde889cfd8e1dabb59c277
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 30 20:34:03 2021 -0400

    release 0.1.1619828897-0b836a21

commit 0b836a2147103cebd8a95a15c9990a7f709c2cff
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 30 20:28:17 2021 -0400

    cleanup

commit f252612873458eb95d7e13a3bde6f7ce01a74c4b
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 30 19:05:42 2021 -0400

    minor log changes

commit ce169220b25004ee4a668aec0c1b5b86bdcbeabf
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 30 18:59:26 2021 -0400

    convert to inmemory servers for observers vs sockets

commit 670fb1be1e2c47b37dcdefce7b4fa194263aa993
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 25 10:54:46 2021 -0400

    release 0.1.1619362422-308de9aa

commit 308de9aa4ea9b5e492d0242cb997ab4b067791c1
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 25 10:53:42 2021 -0400

    cleanup bootstrap errors

commit 46d974df4058b5591b884774b71e971b2a9b3042
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 24 13:07:29 2021 -0400

    test fixes

commit 65c5e19cdab2a840710a4fbc0532d2eb29cbd091
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 24 12:53:09 2021 -0400

    release 0.1.1619283130-0fb4861b

commit 0fb4861b0340d504fc35da5b8844097597cddde3
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 24 12:36:41 2021 -0400

    implement notary synchronization.
    
    - uses a bloomfilter to keep track of seen grants.
    - initially synchronizes pretty rapidly, and as time goes on slows down
      to around once every ~8 hours.

commit 675b010d2c65372f5b57e713fbd4ef7670b03731
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 24 11:01:35 2021 -0400

    notary sync backend

commit 2ae54307ae1363633db25d40b55c565300ffe52b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 17 09:33:26 2021 -0400

    release 0.1.1618662611-a823d1cf

commit a823d1cff072d0a6fc4cf827aa4efe7a2bd9c087
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 17 08:30:11 2021 -0400

    remove comment from notary.proto

commit f46cd5930a3c46d3ebd3bc9b133e47b143904bbc
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 17 08:29:19 2021 -0400

    ensure ssh comment is maintained when modifying the auths files

commit 4ff8b62c8071997adfed95ce5636882f2d24c910
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Apr 15 06:54:08 2021 -0400

    small patch to fix client notary behavior

commit c68ab6832736f514738239b55046820c276289a0
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Apr 14 17:50:27 2021 -0400

    release 0.1.1618436426-8f1db21a

commit 8f1db21aeaebb993ee29b488f7e9c0bcf70cbf07
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Apr 14 17:40:26 2021 -0400

    remove yamux from muxer code

commit a792e6ecb6c3c25ec03fe59f53975f019d68b70d
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Apr 12 08:09:15 2021 -0400

    release 0.1.1618229170-0404ef54

commit 0404ef546abf1f5e8ee464bd5e7b58de49df7251
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Apr 12 07:55:11 2021 -0400

    adjustments to bootstrapping

commit 4070a4ac7a7dabc21026cde014682bbd65e1e7d9
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Apr 12 07:28:18 2021 -0400

    logging

commit 93c0f4bd51d0b431aa1ce0321698dd5d956cf49e
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Apr 12 07:23:09 2021 -0400

    fix archive deploy

commit 8218bd2f931ec80eb9f4ac86127d02ae31318916
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Apr 12 06:21:12 2021 -0400

    client fixes

commit 9273262fd4877b55a28b85a64a4bc8ddd1558dc0
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Apr 12 05:55:09 2021 -0400

    release 0.1.1618221250-481da0a5

commit 481da0a5d3f867bccc05f2ddd1f6d3a47b3c4225
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Apr 12 05:54:10 2021 -0400

    logging

commit 219eef5c64d62db4999215a9e632782292083fb8
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 11 16:06:34 2021 -0400

    release 0.1.1618171503-e6d961b3

commit e6d961b3874634c7387e0043c370fe767e84b6aa
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 19:24:19 2021 -0400

    add authentication requirements to server and quorum

commit bb0b9906da971320ee7ee92f7081c6079cacb7ec
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 11 11:56:55 2021 -0400

    release 0.1.1618154707-9c582067

commit 9c582067c4d5ee3cc21b14d362c20f2922a45edd
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 11 09:07:41 2021 -0400

    test fixes

commit 351c15a83e3ece667b3427187142008d84ccfc8b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 11 08:08:32 2021 -0400

    remove dead code

commit d531e31057dbcd608d372b904ad723df09609c81
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 11 07:34:40 2021 -0400

    test fixes

commit e8dbf80e6601fd47bfdd75f3a39c200185ba0e1d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 11 07:22:30 2021 -0400

    update ginkgo

commit 35138ce1282b68fa9bd9c5fad75dae9d3f4326d5
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 16:07:41 2021 -0400

    release 0.1.1618083316-5b676776

commit 5b676776c252468800b74cb3dbc1e1a158ed4818
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 15:35:16 2021 -0400

    cleanup

commit 461e48f4b6997a5df6640c76e47d7df4922dfbcf
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 15:30:13 2021 -0400

    release 0.1.1618082890-75c879ed

commit 75c879edc6f51a825fa6ec073631fb5abfd59a73
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 15:28:10 2021 -0400

    fix bug in bootstrap

commit bdbfecb168ea2402d59d508b6df94afc7b24f849
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 15:23:29 2021 -0400

    stop bootstrapping when global context is dead

commit 12930637df2fd601c03eb5655b68347904484121
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 14:58:58 2021 -0400

    release 0.1.1618081029-2c4c3904

commit 2c4c390465229682675b6163a8c9a918c18278e4
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 14:57:09 2021 -0400

    allow adjusting the passive checking for the consensus leader

commit f531e70f034e7f8f1a8d7b9db8260bb62f5d86a6
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 14:50:29 2021 -0400

    increase passive checkin to an hour

commit 175d065e8298cd749fd086068f772d3cdb023c2b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 14:45:56 2021 -0400

    release 0.1.1618080225-782b188d

commit 782b188d942c07cccd2800a1e3daae909006ca13
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 14:25:05 2021 -0400

    update aws dns command

commit 590704dfb10b9b302ab7a454e0c1359edc958376
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 12:21:06 2021 -0400

    adjust clustering timeout to a minute

commit 887d470e9c6d59208cb5dbf74a1fe03cfa136a26
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 11:51:52 2021 -0400

    silence muxer logs on failed connections

commit 24e2fa45ea3408a3f21e32d39953032f0f340cbc
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 11:29:51 2021 -0400

    release 0.1.1618068532-55fdd9ad

commit 55fdd9ad5d8c77b01d9f0773e79132b1b00aee02
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 10 11:27:44 2021 -0400

    optimize clustering to use all sources at once

commit 9dac1ac35e00be37cef0471df8b12f7b129f24de
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 9 14:41:01 2021 -0400

    release 0.1.1617993564-6515b9c9

commit 6515b9c979799b35d03c63a366e87882361b454f
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 9 14:16:10 2021 -0400

    allow alternate bind address

commit 4846d2fe6a843bbc081b4077eef2b34569a9de08
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 19:09:18 2021 -0400

    test fixes

commit 083b45a43a04341c9664302f8fab1344d7bc3e0d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 18:39:37 2021 -0400

    test fixes

commit d141d24400d395fc6a80e650deacf021a6dd470b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 13:39:04 2021 -0400

    release 0.1.1617557798-797c7778

commit 797c777852c39ef87e8f822703bf6d285dafc0fb
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 13:36:19 2021 -0400

    fix notifications

commit da74cd74322ab2a6b6902f3f1207b94863f53745
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 11:13:45 2021 -0400

    update development configuration

commit 15a5b751d6ccaf625b63fe30518487c8fe843fdb
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 11:07:49 2021 -0400

    remove logging

commit c83d7672c8949405e8257b37418f69a4136a5e77
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 10:58:24 2021 -0400

    update readme

commit 32b5025d514384a47d6b4bed4faa8ca18e9441bf
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 10:37:58 2021 -0400

    rate limit the certificate requests

commit b863a1e8da875b12f0c59b433bb9c6b819ea2c31
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 10:22:27 2021 -0400

    improve me output

commit 7ad24ed4e7e48de042290f32325375c5e069d2d7
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 10:01:25 2021 -0400

    fix up client for deployments

commit 84da6bc87bfb8900e7fb3fcb175fbc45775365da
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 08:55:50 2021 -0400

    debugging acme

commit 5043e1faadb6f7af31d947caa460d5a80d9840cd
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 08:51:33 2021 -0400

    update ports to muxer port

commit 26bed503004c09c264c5c739cfb58ca8d60dc454
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 4 08:49:15 2021 -0400

    update clustering code

commit 92941c1701b8c49e08f1f3e4af0f7189caccc57e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 3 22:22:04 2021 -0400

    release 0.1.1617502856-5f1fd2a0

commit 5f1fd2a020d754736f86fe5d9e6df2beb536a1a0
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 3 22:20:56 2021 -0400

    test fixes

commit e58a333160f9e2b01bb870b032bddfcf98da622d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 3 21:20:26 2021 -0400

    release 0.1.1617499132-4905df47

commit 4905df4758fd7772d10328f4dc6810e33fedfe3f
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 3 19:29:44 2021 -0400

    update build

commit 2cca8f28a4573be8b9cd0e4d90438f2b79739b58
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 3 18:59:49 2021 -0400

    cleanup discovery field from configuration

commit e69e7c3eb2017b81cbd368db6ab7f306aedc28a4
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 3 18:41:43 2021 -0400

    cleanup raftutil last contact logic

commit 6aa6c95a2f8940be91f83c8e7dc1dd4592ef8981
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 3 13:17:02 2021 -0400

    improve cancel logic to include who cancelled the deploy

commit efbed9ee66546480a7a67c96aa7bad726124ca32
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 3 11:43:59 2021 -0400

    cleanup

commit 7b8eae5d8a7a6bcb6eea3d36ce2d332ac66ff46c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 3 11:21:16 2021 -0400

    update dependencies

commit cd2b634483346ea0bec048682a7345c07fffdf7d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Mar 7 12:53:46 2021 -0500

    agent reboot
    
    - collapses operational complexity of the agent down to a single port.
    - makes autocertificate code more secure.

commit 8986e355d36f0f5b2cd50a44835b5532a0d14ce9
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 20 11:33:41 2021 -0400

    muxer implementation

commit c734123fbd497887916519712139fa2b6855ecaa
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 20 09:42:45 2021 -0400

    update lego, work towards muxer

commit b8551e2ba84512b9920b5fc956f4720b6dfe0c9c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Mar 7 06:15:31 2021 -0500

    release 0.1.1615066684-1a4acad7

commit 1a4acad73737b5242926dca883f9e356ed81866b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 6 16:38:04 2021 -0500

    debugging logs

commit d4dcfac96cb93212ada65e70bb1ee148d68c9dbc
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 6 15:04:17 2021 -0500

    enable quorum command

commit 4b20c7cf5b0aa5d40717d2ecfe21a4fd57d339ef
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 6 14:59:45 2021 -0500

    add quorum members to info

commit 57f316bd2a5ca6501cf1a4220667c37ac501ea89
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 6 14:59:20 2021 -0500

    add quorum members to quorum info

commit 91b4991b431bff2488d2925de5f33ec154a14d23
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 6 14:58:58 2021 -0500

    update dependencies

commit 97cb0c913413f7b00eac7cbb9354acb10f5a9fac
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 6 13:46:49 2021 -0500

    add info to agentctl

commit de2b261498dd3d55fe41118c13a14a866927084e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 6 13:01:10 2021 -0500

    clustering debug

commit c4cab818d0f42c71765cf488cc0ace944a0d8133
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Mar 4 04:41:56 2021 -0500

    release 0.1.1614850675-164713c6

commit 164713c6dc1b45c83d2c9eec69f446cf6252fa23
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Mar 4 04:37:55 2021 -0500

    improve debug log

commit 5576e989d319c4b5758dbcd77e21a56ec0ff5324
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Mar 4 04:33:10 2021 -0500

    tickle raft overlay when quoromOnly check fails.
    
    i suspect the raft cluster was shutdown, and then locked into a pending
    state. this tickle should reboot it.

commit cac63ab07e7d0ffe546f07395bf9e1a86d144161
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Mar 4 03:51:49 2021 -0500

    implement additional debugging

commit 4dc3196cb04667666cdc753c17ef7c59e0c63d73
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Mar 4 03:51:20 2021 -0500

    remove packagekit

commit 3401564a2a12eba385775ff40785e801bc883b80
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 30 11:49:30 2021 -0500

    release 0.1.1612023911-bf556b99

commit bf556b995b1a7987d771464a881af4a774bd1b1b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 30 11:25:11 2021 -0500

    remove empty test

commit 5bcacb70d4236518ec5168d8112b13a13e82b0fc
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 30 11:04:08 2021 -0500

    remove timed transition from condition transition

commit 21bd47ab9939ca6424c9a0330d870844a99fb1a5
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 30 10:20:44 2021 -0500

    release 0.1.1611999253-6fff3625

commit 6fff36259e51caffa9a73d29b20f0582c178a156
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 30 04:34:13 2021 -0500

    attempt to resolve multiple passive state routines
    
    we seem to spawn multiple raft instances over time.
    leading to multiple conditionTransitions and background routines.
    
    this seems to be related to the fact the state machine was blocking
    on the protocol's context, which isnt cancelled until process shutdown.
    
    meaning everytime a node would be promoted into the cluster and then
    demoted it wouldn't reset the raft cluster properly. specifically the
    network connections wouldn't be reset. this buildup would lead to a
    gradual resource exhaustion of file handles.

commit a52b137fbc3edbaeecc64efc01140f0d41e1086f
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 20 10:22:44 2020 -0500

    cleanup deploy and bootstrap tests

commit ae1eac85f1550fa27e059675ade68127d77364c6
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 20 09:11:47 2020 -0500

    update client configuration to contain auth keys

commit 7d5ec33c59de17b2737d9a9cb2605e1961bf28dd
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 19 19:02:49 2020 -0500

    release 0.1.1608422512-9cfe2866

commit 9cfe2866c7a5eba318ef4bd7bf3ae33632a814c1
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 19 18:31:33 2020 -0500

    change golang build version

commit 6ec879c3de26d043824d4b83bce74acfec3acead
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 19 18:21:59 2020 -0500

    update build

commit 5e2e713cb42f2562476e3eff1d910d6deee94d4e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 19 15:58:38 2020 -0500

    add auth key to me init function

commit d59fcbb61c37eee6027d6f1e69cfa7a2706cd512
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 19 14:29:24 2020 -0500

    release 0.1.1608406076-21f1c9bd

commit 21f1c9bde48c9256cec12700fe6adc27e279e484
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 19 14:27:56 2020 -0500

    update yaegi

commit e3ffc73e188336dfcf829d0acc8f31723dbb25fa
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 19 09:30:36 2020 -0500

    release 0.1.1608388182-b5bafcd5

commit b5bafcd5359b7a8ecaa3dcd2dc819b3b5b9c5d65
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 19 09:29:42 2020 -0500

    update release

commit 940b05e12b0de5bd59d203e54226d9a3b4ee56b6
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 19 09:21:19 2020 -0500

    implement authorization file from deployments

commit 00ff6917d7973f1647b632a57a43d7123783825d
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Dec 18 16:22:56 2020 -0500

    remove unused sync code

commit 0709d8a45b72df5acd1307f55eca09cae7c7aaaf
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Dec 18 15:49:43 2020 -0500

    remove deprecated wal decoding

commit 8d7c7a9ecaf3c75adbcebb321017e205be3c8593
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Dec 18 14:51:52 2020 -0500

    update raft

commit b757a4926a663b244ed314480a64e3cb545a3299
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Sep 27 06:09:37 2020 -0400

    cleanup logs

commit 08b7054e73313e3aaa3590d04ae18ade1db82bfd
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Sep 27 05:58:18 2020 -0400

    standardize pebble setup for test environment

commit e246ec0a0eb80dd134e88127dd2ec660e3c7117e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Sep 27 05:13:34 2020 -0400

    log fix

commit 521de13dacad403e3d11695ad12249e89bd9ef1b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Sep 27 05:11:52 2020 -0400

    Revert "Revert "update yaegi""
    
    This reverts commit e8e2b1be092194e3308430f6827c615a8838776a.

commit cd0a7ffe08332429651b75ccdf5bab3144ac5b10
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Sep 27 05:11:34 2020 -0400

    log tests

commit fee0c9480f4e951f5bdbb4a30a9084a006cb07eb
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 26 15:49:39 2020 -0400

    release 0.1.1601149727-7eb04af3

commit 7eb04af3523dd3b783376d68e5a1e99e447c9dee
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 26 15:45:56 2020 -0400

    fix build

commit d6d3413a973bef5ac1499172716865ce01b4993d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 26 13:07:44 2020 -0400

    debug

commit e8e2b1be092194e3308430f6827c615a8838776a
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 26 13:54:23 2020 -0400

    Revert "update yaegi"
    
    This reverts commit 97f388efaae92d8975773e3b1fb177a7e10701f6.

commit e11af3b9b2f66e87693a2eab66a15e5bfaedbac5
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 26 10:21:11 2020 -0400

    add logging for passive cluster state reset

commit 96821177c30b6f65d52dd851fb07b5a35067b035
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 26 09:48:10 2020 -0400

    release 0.1.1601128035-e06a20e6

commit e06a20e6b03578b01cb532103aa570557a30ed61
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 26 09:42:20 2020 -0400

    update build to ubuntu 20.04 LTS

commit b6b3c66fd6926c08cab6c83186321b27a7f0c78e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 26 08:56:35 2020 -0400

    update makefile

commit ac119229c00b7fbc2efcadec6f8dc793b43b0d26
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 26 08:48:50 2020 -0400

    persist agent name to disk on first start.
    
    possible fix for duplicate address error in raft.

commit 97f388efaae92d8975773e3b1fb177a7e10701f6
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Sep 26 06:34:59 2020 -0400

    update yaegi

commit 5b6c0458c857b759ae07abcf009a451eff3b0239
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun May 17 07:52:47 2020 -0400

    release 0.1.1589714724-c8466fe1

commit c8466fe149fd6a9625e49a8c353e8c8b05c3365e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun May 17 07:25:24 2020 -0400

    release 0.1.1589714697-1ac4e282

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
