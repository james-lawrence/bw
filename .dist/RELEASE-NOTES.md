commit ce0b8e9daa2a48c5b84c179d67597205ef29fef9
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Nov 10 07:20:36 2018 -0500

    release updates

commit 1b237389baac6924f42aaa5ab22a456f2e35f5c1
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Nov 8 20:39:46 2018 -0500

    improves deploy behaviour.
    
    - now exits when deploy completes.
    - colored output.
    - now grabs logs when a deploy fails.

commit c2f4d3577cb9de1960fa8bf0b97d3e0edbb4b852
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Nov 8 05:18:12 2018 -0500

    add logs endpoint for retrieving deployment logs

commit 567c50f5300c426d36cf26f2e412c2265df93109
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Sep 23 06:49:00 2018 -0400

    remove client creds generation from self-signed

commit bfed395f0043a67196c9a9143589402343113b06
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Sep 12 18:15:55 2018 -0400

    change secret to cluster tokens, allows for multiple different tokens

commit 43245c844f8657f1bd94c2bcb235e2ac74ef1b67
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Sep 11 06:33:30 2018 -0400

    reeanble bootstrap errors

commit 432ff8fa38737047ed69f64445f0b3205431ed78
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Sep 11 06:33:14 2018 -0400

    log cleanups

commit f7e8842f0d892a84fd96e772e82eefee79f6ad04
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Sep 7 23:06:52 2018 -0400

    move spike to its own root level directory

commit c164a1571a84e469d9df526a60f712901b46bac6
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Sep 7 22:44:29 2018 -0400

    bootstrap fixes

commit d16f65d27ba591d553733f72ee1935e62415d36c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 19 13:00:00 2018 -0400

    first pass at resolving the bootstrap mid deploy race condition

commit 5c076bfa630f802cf6123a9c02e30ca35831a64b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 19 15:00:21 2018 -0400

    remove logging

commit 124d921eebd0928462f0b4f1fa5ecc6fb17b63cf
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 19 15:00:04 2018 -0400

    clean up some connection leaks in the client

commit e690bb9608133812b33a50951e4f6ec071abad1b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 19 14:43:34 2018 -0400

    update raft library with a bunch of stability patches

commit 8ea387e509ccd1e37442ec6819713ada13ef6544
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 19 12:58:20 2018 -0400

    remove extraneous log message

commit aed5545e0f112f1fff2fdedf3a0f395aa6f334d6
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 19 12:57:32 2018 -0400

    cleanup error message

commit 7b0bf9c82c65c231fec0965ac287c433742d4e28
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 19 12:24:04 2018 -0400

    deploy commands should also contain the deployment options

commit efef0b1f7cab7879fa0dca4c0dbfe84fcf195fc2
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 19 12:22:58 2018 -0400

    an active leader should not leave quorum

commit f99e07f103efb6deb1c1cf6c4d02e98b87e01dc1
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 19 11:10:55 2018 -0400

    progress towards resolving the mid-deploy bootstrap race conditions

commit aaf640884824c14ec5fb75c2403048fc7630eea6
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 19 08:53:08 2018 -0400

    small adjustment to error handling during bootstrap

commit a0eca7849c2a4902757154cd0731b6b7bd67d974
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 19 08:50:59 2018 -0400

    change fqdn resolution to not error out on any given hostname

commit f3ea2bf7ce021eeb0b46f33cdc04950408bd3ae7
Author: James <james-lawrence@users.noreply.github.com>
Date:   Mon Aug 13 16:41:01 2018 -0400

    Update context.go

commit 4bf2263567215b41a3160bc79217b3063e0ea68e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 12 15:19:44 2018 -0400

    enable local directives

commit af42fe3926a174d7e990dc2cfd48feb476014cc3
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 12 12:45:55 2018 -0400

    change environment variables for dns

commit 0c86e909cf36c12a4f394c83c160d3278768050e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 12 10:21:05 2018 -0400

    update systemd unit files

commit 4ca9588b76b13083f5f6784b753fefbad3577486
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 12 10:18:44 2018 -0400

    update release/dist script to include example systemd files

commit d29b310e49f3f56a5146abd16618935bd6ee1b57
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 12 09:30:25 2018 -0400

    update docs

commit 790e1129856406a1c261e6e2252a8bc95093b6b0
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Aug 11 17:17:00 2018 -0400

    initial work for google cloud dns

commit e593730f2420fa89128ece17097b64090889fa09
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Aug 11 20:58:15 2018 -0400

    move default agent configuration directory

commit b9fb9b19fc1ed4adf62d743a1e89478ded1c7a8d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Aug 11 18:33:54 2018 -0400

    move notification configuration into same directory as agent config

commit 65ec81edbd20384441f18ec297879a0404fc9097
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Aug 5 14:29:16 2018 -0400

    implement a more reliable logic for message dispatching

commit 12dd351f0530f9bca529fa2b9cf38d146ed0aa5c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Aug 4 19:02:32 2018 -0400

    bump up timeouts for raft transport layer

commit 4f3d2a0a69334e8b73d23a6dc1357bc0f0ed225e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Aug 4 18:15:22 2018 -0400

    minor changes to raft bootstrap and configuration
    
    hope to improve reliability of the recovery processes
    for the cluster.

commit 2980f7801f9d5230abe3facd780a667c4651950d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Aug 4 15:27:52 2018 -0400

    reliable quorum management

commit 3b58e1b5d224a992f083b62f148e28d8e7510d11
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Aug 4 14:57:30 2018 -0400

    update raft library with misc fixes

commit eb267e9bfe45f7ac87c775b3d06bb6387ccf267d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Aug 4 07:06:30 2018 -0400

    apply event ids to another message type

commit 7004a79d3372847d2b815ffee92ea2b614e1fea3
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Aug 3 21:02:45 2018 -0400

    automatically restart an active deployment when leader dies mid deploy

commit 4a28a1f817178fd7c8acd50d513e870ad7ed9612
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Aug 3 16:19:50 2018 -0400

    move metadata into agent protocol

commit 0ab6ddd2011ce2ba18e8c015f390a01bedcfb1ca
Author: James Lawrence <james@talla.com>
Date:   Fri May 25 11:43:12 2018 -0400

    fixes for local deployment

commit 28e9f8b5329f983818146493134b6f8e5a2a777e
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Aug 3 13:44:17 2018 -0400

    properly shutdown info

commit 934336dbab7ea172803356ae855dcd605609af29
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Aug 3 13:12:04 2018 -0400

    finish changes to notification updates

commit a856ea19cedcde94d48e8a844edfecfe99b921fa
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jul 31 17:45:31 2018 -0400

    more robust notifications, automatically reconnect instead of crashing

commit 3e12c2410a50e865d10fedb0d58bd2a59950cb62
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jul 14 07:47:53 2018 -0400

    bugfixes and misc cleanup.
    
    - cleanup: remove bootstrap timeout configuration option
    use bootstrap attempts to limit number of attempts and the timeout in
    the deploy metadata to limit the time for the deploy itself.
    
    - bugfix: do not attempt to add peers with empty server addresses to the
    configuration.
    
    - bugfix: cleanup edge cases around deploy timeouts, and make the default
    timeout consistant throughout the codebase.

commit fa102cfe7be46082792dc25866439a08762d389d
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Jul 10 19:58:44 2018 -0400

    fix dialing self as part of the quorum, does not make sense

commit 4f365d6fbf2cc130323efec9f8ecbce80bb447be
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jul 1 15:02:59 2018 -0400

    minor command cleanup

commit 405257346f03b70272c5b56f03723ec3e693da4c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jul 1 14:53:42 2018 -0400

    improvements around event streams
    
    - automatically reconnect when a connection is lost.
    - detect client side disconnects properly.

commit 5a0da01f3895f417416f8b2bdc552e1bbc7bc68b
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jun 22 17:17:10 2018 -0400

    use configured timeout for local deploy

commit 762d39cd6c2a36aff05157287267dfd18ea54601
Author: James Lawrence <james@talla.com>
Date:   Thu Jun 14 14:11:04 2018 -0400

    properly use environment for local  deploys

commit 43532b2f365aecd4240ada8f424a38c384840523
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jun 10 14:44:05 2018 -0400

    start ACME2 testing, update readme and make configuration generation work

commit 3a9d577f88932313c5f7af450c1968d166cd5f8f
Author: James Lawrence <james@talla.com>
Date:   Mon Jun 4 10:29:16 2018 -0400

    reintroduce deployspace CLI option as deprecated

commit ece92d9245a02d4dc75cfbb2c7a3b44389db6ce7
Author: James Lawrence <james@talla.com>
Date:   Fri May 25 11:43:12 2018 -0400

    fixes for local deployment

commit d2a2758cf99c720695218ada9311223e740e7fd6
Author: James Lawrence <james@talla.com>
Date:   Fri May 25 11:43:12 2018 -0400

    fixes for local deployment

commit 19d20b9b35d762050f7af584719f08c5a6fd14c5
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun May 6 10:12:23 2018 -0400

    implement vault PKI source, add ability to automatically refresh credentials

commit 0c83a36cab8c51238cd2ce632483403f147ac38a
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat May 5 08:29:58 2018 -0400

    ensure vault credentials gets the correct directory

commit f5004ed18359485aa686460754a69ce5a696b849
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat May 5 08:15:15 2018 -0400

    move logic to retrieve TLS certificates from vault into the credentialscache package

commit 8d63086eb8c0c46da7447a03826d497aee582e44
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat May 5 07:39:41 2018 -0400

    switch client connection to use credentialscache

commit a256a49445eced6a1da6a9ebf6c14df611966dfe
Author: James Lawrence <james@talla.com>
Date:   Sat Apr 28 11:05:39 2018 -0400

    improve some timeout handling by the server.
    
    - client defaults are used, i.e. doesn't timeout, letting the server
    control the behaviour.
    - server timeouts drastically reduced. exposing some issues with the
    proxy service not backgrounding the bulk of the deploy.

commit 5620a4d9d0cd83e777c1cf1b00926f61a26ebcab
Author: James Lawrence <james@talla.com>
Date:   Sat Apr 28 08:35:01 2018 -0400

    small adjustments to deploy timeouts

commit a54fd8acde10631ed7d1a83521c7925e3ff06f5b
Merge: 7974917e 130786c6
Author: James <james-lawrence@users.noreply.github.com>
Date:   Fri Apr 27 19:46:12 2018 -0400

    Merge pull request #10 from james-lawrence/refactor-dispatching
    
    Refactor dispatching

commit 130786c66132b9e4d9991096af0a3e2a50f6c8f3
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 27 19:36:48 2018 -0400

    clean up commented code

commit 99bd55d63057342679a532f088d267b4014e0477
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 27 19:35:31 2018 -0400

    prevent filtered deploys from running when no matches are found

commit efbb06d853f6c4f851209f4bc6e1e24fb94f9717
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 27 19:27:20 2018 -0400

    properly return errors from invoking dispatch on the state machine

commit f54db0527fec6a7709bea9672f2286addc36247c
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Apr 27 17:45:28 2018 -0400

    fix dispatch messaging to be delivered for all quorum members, not just the leader

commit 7974917eabcd0fe693d2f639b871148fa6ccb450
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Apr 25 18:18:02 2018 -0400

    log out when dispatch to clients fails

commit 61bb5be453c23d5b467b875af815543b4224e552
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 22 10:08:53 2018 -0400

    remove s3 storage code

commit d2556e73d16920b34f1bfbe55491ec3327389b8f
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 15 09:01:00 2018 -0400

    small fix for conn dispatcher

commit c32cd9c21ec2f4b9b0c4d97cf89f015f83efcdc9
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 14 18:44:56 2018 -0400

    implement credential manager

commit 07cff3311922729d4fa3d93f844631007b30116d
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 14 12:56:59 2018 -0400

    test fixes

commit 9baae3a30ef40d697663dced9a85d47c3fb79fb8
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Apr 14 12:12:19 2018 -0400

    initial cancel work

commit 570d3b211608c3aeecf27d1d1c9adf2a260f8598
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 8 20:43:40 2018 -0400

    usability updates to info command

commit c6d5808cb600c8a2b809aa5c3b45d44c94f98181
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Mar 2 22:58:18 2018 -0500

    better quorum

commit 0f3ccd24dd8aedcf09b416615aef99b4c914f9c9
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Apr 8 09:47:36 2018 -0400

    fix deploys not stopping after failures

commit 13340251dcd9dfcbfeab4683e7599f69a5367591
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 31 11:06:22 2018 -0400

    improve health check for awselb

commit 9b0600a50b0d8066788f5ccec12ee887ed5c7576
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 31 11:06:03 2018 -0400

    remove dead code

commit ba218a1b194cc58da7aa003dba6b0ae950274bdf
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 31 11:04:56 2018 -0400

    add user information to deployment archive

commit 090c84053b7e246dd2195ef181d80a2ad667c706
Author: James Lawrence <james@talla.com>
Date:   Sun Mar 25 09:34:06 2018 -0400

    bootstrap should not return success when a deployment fails

commit de8196429080676e32e4bc94e0aec8f6ecf5421d
Author: James Lawrence <james@talla.com>
Date:   Sat Mar 24 07:44:17 2018 -0400

    ignore instances not part of an autoscaling group

commit 015bb71b7a293230ffc26dc6659761c9ad5ecdc3
Author: James Lawrence <james@talla.com>
Date:   Sat Mar 24 07:25:38 2018 -0400

    ignore elb if not member of autoscaling group

commit 9aec42d85203c77ff5fb6c9c5c0efa758e759999
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Mar 21 19:43:39 2018 -0400

    Revert "split client from agent binary"
    
    This reverts commit 689e6855f29ab52784e864d26409562985eebdbd.

commit 689e6855f29ab52784e864d26409562985eebdbd
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Mar 20 20:58:51 2018 -0400

    split client from agent binary

commit fbca5cce945ecdc0a5990aef8851fba12a560cf9
Author: James Lawrence <jlawrence@datto.com>
Date:   Tue Mar 20 14:47:26 2018 -0400

    allow compilation on macosx

commit cf2fbe87bece3ccf899f2b1e6f08feb988b7f343
Author: James Lawrence <james@talla.com>
Date:   Sat Mar 17 08:51:15 2018 -0400

    misc cleanup for setup

commit b13925e5469d930a3d52295c100203634fe75d42
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Mar 16 18:33:22 2018 -0400

    do not push bootstrap events through the normal dispatcher

commit 0e6d5574595acaa29c10e8849de1e1b88a14ea3e
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Mar 16 17:13:44 2018 -0400

    remove unnecessary logging

commit f3f087c279fc54ce2c188d4bc43ea04c7b47b64b
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Mar 16 17:06:12 2018 -0400

    support locally deploying based on a deployspace

commit d0be47880eeeed66deeda959b1759edc5346debe
Author: James Lawrence <james@talla.com>
Date:   Fri Mar 16 10:32:39 2018 -0400

    always return a quorum client from connect

commit b20d2992aa29c16465b71cf96259059dc818bb31
Author: James Lawrence <james@talla.com>
Date:   Fri Mar 16 06:49:40 2018 -0400

    updates to notify

commit 7f9f0d6d89694ac3f9254529481f8c9e4bac099c
Author: James Lawrence <james@talla.com>
Date:   Fri Mar 16 06:29:47 2018 -0400

    do not set last contact when a vote request is received

commit 07af913145cd93e6dd00367d9d6abb9d00b5af16
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Mar 4 21:16:53 2018 -0500

    better tests for the raft overlay

commit b9e27e3cb95cc458cc56ec16098923193f7198b4
Author: James Lawrence <jlawrence@datto.com>
Date:   Wed Mar 7 09:15:27 2018 -0500

    fix seg fault

commit 9f5613583a4cc9d05a42f691bd10ea10baf16233
Author: James Lawrence <james@talla.com>
Date:   Sat Mar 3 12:19:19 2018 -0500

    Fix issues with backwards compatability, and an edge cause
    
    found during during a deploy.

commit d57388a931668a62ecbfc368fe88f3bd7a470058
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 3 11:15:37 2018 -0500

    update readme

commit a78b0f65ba229fb4db43ebb52942e6ae426b4fe7
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 3 11:15:05 2018 -0500

    cleanup notify

commit e46d003bdeb9a7fd80f165984a01e7cf0f73391c
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Mar 3 11:03:41 2018 -0500

    create the dialer concept

commit 1d7169d62fb370de66dfa8d70cfb88a1bac31285
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Feb 26 21:23:04 2018 -0500

    add post deploy bus, cleanup comments/old code

commit 38f0933e14628614aceb62c1ac0c79f43d5f8ace
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Feb 25 20:26:33 2018 -0500

    switch back to using local coordinator during bootstrap, less chance of causing a deployment lock if there is an issue with the bootstrap

commit f324c75029c36e06fa43fe13aa94918d61690777
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Feb 25 20:02:16 2018 -0500

    move completed event to be based on message events

commit 2e263b8cbe48045f37a86b9e32a6f038f07efcc4
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Feb 25 19:17:34 2018 -0500

    rework protocol to be cleaner

commit c2ac3ee26e89c733d773e790a8d79aacef844eba
Author: James Lawrence <james@talla.com>
Date:   Wed Feb 21 15:56:58 2018 -0500

    properly detect the configuration directory, even when its empty

commit 53fd73542c9b92f3113a6eb2534fee34d521e5b2
Author: James Lawrence <james@talla.com>
Date:   Wed Feb 21 15:51:04 2018 -0500

    fix vault credential generation (write to the correct filepath....)

commit c2a287c9204692415e2aedfbc361191a4fbfe5cf
Author: James Lawrence <james@talla.com>
Date:   Sun Feb 18 19:02:34 2018 -0500

    update dependencies

commit 3c3663a45d614973aaf18cfb67b823b279e0e524
Author: James Lawrence <james@talla.com>
Date:   Sun Feb 18 10:30:12 2018 -0500

    add license

commit 1d6934910213ac35d5bebd7e99d60884b55f2d3a
Author: James Lawrence <james@talla.com>
Date:   Sun Feb 18 10:14:43 2018 -0500

    start documentation process

commit afe8b608bc44fc244f7671c13cf11aa718c341d8
Author: James Lawrence <james@talla.com>
Date:   Sun Feb 18 09:03:02 2018 -0500

    add vault integrations for public keys

commit 7a62d54f9ef16bcc2c9e3db81b6eed5a5e05e215
Author: James Lawrence <james@talla.com>
Date:   Sat Feb 17 16:41:13 2018 -0500

    rename variable

commit 1978a944bc9d7c8ef5c0b031e9ae09d93c78adfe
Author: James Lawrence <james@talla.com>
Date:   Sat Feb 17 16:38:33 2018 -0500

    fix leaving quorum
    
    when a peer downgrades into passive mode because leader grace period
    has expired, we need to immediately check if a promotion is needed
    and not wait for another (potentially never occurring) cluster event.

commit e846c43a3b37e18f80fd89122f0a863cd78de39d
Author: James Lawrence <james@talla.com>
Date:   Sat Feb 17 16:19:43 2018 -0500

    fix MaybeLog file location in output

commit b80d3a55420dd0e321bac5169e014d5b7474d026
Author: James Lawrence <james@talla.com>
Date:   Sat Feb 17 16:12:04 2018 -0500

    default leader to none if missing

commit f14a2dc6430e2be9ea881fb832df12cd1456e0f2
Author: James Lawrence <james@talla.com>
Date:   Sat Feb 17 15:08:36 2018 -0500

    search all peers for leader node

commit a3c1fbfb779c92f6a290ac81813e20c50ec2aa27
Author: James Lawrence <james@talla.com>
Date:   Sat Feb 17 14:01:17 2018 -0500

    clustering improvements, bootstrap improvements

commit 71a150bf73a77467f6874809c4734412199d301e
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Feb 17 00:44:41 2018 -0500

    torrent storage improvements, attempting to reduce failures when downloading

commit aab4267f7dbb0158ecc684ed8f0e928f11e5c27e
Author: James Lawrence <james@talla.com>
Date:   Sun Feb 4 09:35:03 2018 -0500

    log the notification configurations on startup

commit 9a91cef6c0b0b4d53a4548daead8da1bb3f19d5a
Author: James Lawrence <james@talla.com>
Date:   Sun Feb 4 09:09:27 2018 -0500

    fix default configuration for notifications

commit 02d838662f7993173c4f420b95ba61436d8b3369
Author: James Lawrence <james@talla.com>
Date:   Sun Feb 4 07:47:40 2018 -0500

    update todo

commit 5c8cc7d9dfc43bfdd50157d6e9d3748a6712568a
Author: James Lawrence <james@talla.com>
Date:   Sun Feb 4 07:37:24 2018 -0500

    working notifications

commit 1f6ff964c40d5366bb2a15014d29e77467a738e2
Author: James Lawrence <james@talla.com>
Date:   Sun Feb 4 05:20:37 2018 -0500

    working initial notify command

commit cfaf0ae51d6166635bdbe524d55ab5fd593084e7
Author: James Lawrence <james@talla.com>
Date:   Sat Feb 3 17:15:57 2018 -0500

    work in progress towards notifications

commit abab3e2f23a31b94fae965ee088b3716f3e16ae2
Author: James Lawrence <james@talla.com>
Date:   Sat Feb 3 17:13:57 2018 -0500

    work towards notifications

commit 7ad4d9fe53c9463ac34b2d27fc82f4749563f9ae
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Jan 28 13:16:23 2018 -0500

    protocol updates
    
    - cleaner passing of deployment options through the stack.
    - remove timeout from the state machine. better to push the timeout into
    the individual agents.

commit 6e57ed13fb2b6a940f845b1e8c4e40637fec45d3
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jan 26 20:45:57 2018 -0500

    add deadline for deploys, cleanup log ux

commit 6295ea1d3c77b3e32df7b00668740030a17bc452
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 20 08:06:03 2018 -0500

    test change

commit 8fc39cc7cf9fcc9aef48091cdb39664977149747
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Jan 19 15:39:21 2018 -0500

    fixes/updates for bwfs

commit a8e0fae50d99e2f3caf7a9de3d1865ca429b791b
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Jan 13 12:53:39 2018 -0500

    ensure only a single active deploy at any point

commit 5a47b8b525150ac10eb6db3cdd62c20b24820595
Author: James Lawrence <james@talla.com>
Date:   Sat Jan 6 13:38:53 2018 -0500

    properly leave failed deployments available on disk

commit ff77c226ecf0ba19f28ee1878a35294b42e28912
Author: James Lawrence <james@talla.com>
Date:   Sat Jan 6 11:10:51 2018 -0500

    cleanup environment quoting

commit d64292031634f4481f8263af62db95cf1d64c89a
Author: James Lawrence <james@talla.com>
Date:   Sat Jan 6 09:18:21 2018 -0500

    correctly set working directory for shell commands

commit 4a9d88d4d597f2b4102036135b8305165a980a3a
Author: James Lawrence <james@talla.com>
Date:   Sat Jan 6 08:01:27 2018 -0500

    improvements to filesystem directive

commit 251d1f255ddb830dd0d0fb44e58c0822a4a2d5a3
Author: James Lawrence <jljatone@gmail.com>
Date:   Mon Jan 1 18:59:45 2018 -0500

    allow the agent config to specify environment variables

commit 6742c1807d35d57e9ca3564018ca9fc94b75e1d6
Author: James Lawrence <jljatone@gmail.com>
Date:   Sat Dec 30 18:57:56 2017 -0500

    packagekit cleanup

commit 08e5f94030aa138531c39d436f71093d4741bde8
Author: James Lawrence <james@talla.com>
Date:   Fri Dec 29 19:29:05 2017 -0500

    implement elbv1 attach/detach instances

commit 4a0cf78399aacba149d029ffb5ec93c8c94f8a2c
Author: James Lawrence <james@talla.com>
Date:   Fri Dec 29 17:40:55 2017 -0500

    package kit cleanup

commit a91d3859f0fa6bc440669d8777158259d96ef7ab
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Dec 29 00:58:37 2017 -0500

    use DNS peering source based off the TLS server name

commit 21e4b8ef87c463d86a7bbc09096d003084273184
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Dec 29 00:51:19 2017 -0500

    add AWS utility command for bootstrapping a cluster via dns

commit c42cfbc031eed9b83deda8086c04379c43909e3f
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 12:31:51 2017 -0500

    remove yaml v1

commit 3c0089082895c2bead219e532d4f8e2014bdb88f
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 12:11:43 2017 -0500

    add dns peering strategy

commit 6f912780864ef35289fc12a59edb26a145239610
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 11:38:46 2017 -0500

    clean up capabilities bitfield

commit 08399eb8ef8535e27a0e78dbaa1a9d96e45dc3ef
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 10:49:12 2017 -0500

    cleanup of agent status

commit 9fd42d1751b9b09d68a075228dd0845f80ce0a55
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 09:57:25 2017 -0500

    update dependencies

commit 838fb88035c139785ed6dc11b29d073fb491bc19
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 09:41:18 2017 -0500

    ux improvements

commit 96d02918eff3a633ed901b8cbd3c362edba318ca
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 09:21:28 2017 -0500

    ux improvements

commit 37ca6fc3a97054a148dfda6df7247c9c2df3e26a
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 08:37:18 2017 -0500

    move aws bootstrap configuration into config file

commit c5eb581cf84635b4e1b981e44123b6486a5adf2b
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 08:08:19 2017 -0500

    remove glide, and update dependencies

commit 28162e81c9666f51af913b968f39fbf5da2de88b
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 07:52:11 2017 -0500

    remove spike command

commit d0e1d52c5634a1e15acb4795a47e6ae288a5ee71
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 07:44:40 2017 -0500

    more conservative torrent configuration

commit e075c4ec8ce355d11214dd4be2a51cd3d1af9f8f
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 07:28:23 2017 -0500

    torrent storage removed unnecessary code

commit 8b09217519993ad2036cbbb0da9d58495613f17d
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 04:58:53 2017 -0500

    remove file implementation from storage

commit bd7afebe52aa30503ae278155ae713838a39ae68
Author: James Lawrence <jljatone@gmail.com>
Date:   Thu Dec 28 04:39:37 2017 -0500

    torrent code cleanup

commit d49a68240380d6e354c34d85b30926cd75a7ae30
Author: James Lawrence <jljatone@gmail.com>
Date:   Wed Dec 27 17:24:21 2017 -0500

    basic working torrent storage

commit 2255ed3b308d1fc76c60d852072c710826b105e3
Author: James Lawrence <jljatone@gmail.com>
Date:   Tue Dec 26 17:26:38 2017 -0500

    misc cleanup of configuration, moved more options into the configuration file

commit 4533b0280fce34fe6ddb5f62217573b030af87ac
Author: James Lawrence <james@talla.com>
Date:   Mon Dec 25 13:04:42 2017 -0500

    misc cleanup of logging and configuration, added new TODOs

commit 3eb0b3598aef5d7deb0f7b7f3b0c9226a1663500
Author: James Lawrence <james@talla.com>
Date:   Sat Dec 23 13:03:47 2017 -0500

    fix deployment

commit 3c975e72f7002e12abcf9bba75fa6bb55db05960
Author: James Lawrence <james@talla.com>
Date:   Sat Dec 23 12:14:21 2017 -0500

    further cleanups

commit 275248a9d90490cc2be2b4e3ff0f6f4072fdad92
Author: James Lawrence <james@talla.com>
Date:   Sat Dec 23 10:40:34 2017 -0500

    change order of deployment stage

commit 725d640adb4f3f07797a87faf5f0a6f20432aabe
Author: James Lawrence <james@talla.com>
Date:   Sat Dec 23 10:37:09 2017 -0500

    api cleanup

commit 9362eb1a76117132d2a16100d207cfb4486fe856
Merge: 5521ab5b f137dcb5
Author: James Lawrence <james@talla.com>
Date:   Sat Dec 23 08:53:37 2017 -0500

    Merge branch 'test-dir'

commit f137dcb5c92a46e334b50a60dd38c175328989d7
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Dec 22 09:38:53 2017 -0500

    test directory

commit 5521ab5bd477c615e64fd6c26fae18c1888218bf
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 3 13:10:16 2017 -0500

    add .dist directory to git

commit 12a4a024ae148d92b92fc083ba2d58adfbf65e44
Author: James Lawrence <jljatone@gmail.com>
Date:   Sun Dec 3 13:08:09 2017 -0500

    add additional tests for quorum

commit ce2d3f9081939f91e22c246323b7279d431cafd8
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Nov 17 17:14:11 2017 -0500

    combine downloads and uploads into a single storage package

commit 4de3016038dd07b9477ff9a50829ea24ec7c9527
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Nov 17 16:58:31 2017 -0500

    fixup default agent config

commit 5f809d89b3003d2d215e7749fbc880b499f195bd
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Nov 17 16:50:03 2017 -0500

    refactor quorum details into their own package

commit 1a351ad278122214ef421f213e4a6ecfb1c9557e
Author: James Lawrence <jljatone@gmail.com>
Date:   Fri Nov 17 14:59:27 2017 -0500

    update dependencies and config
