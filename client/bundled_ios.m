//go:build bundle && cgo

#import <UIKit/UIKit.h>
#import <WebKit/WebKit.h>
#import <Security/Security.h>

// Wails v3.0.0 uses this controller as its WKNavigationDelegate.
@interface WailsViewController : UIViewController <WKNavigationDelegate>
@end

static NSData *loopbackCertificate;

void pinBetterMusicLoopbackCertificate(const void *der, unsigned long length) {
    loopbackCertificate = [[NSData alloc] initWithBytes:der length:length];
}

@implementation WailsViewController (BetterMusicLoopbackTrust)
- (void)webView:(WKWebView *)webView
    didReceiveAuthenticationChallenge:(NSURLAuthenticationChallenge *)challenge
    completionHandler:(void (^)(NSURLSessionAuthChallengeDisposition, NSURLCredential *))complete {
    NSURLProtectionSpace *space = challenge.protectionSpace;
    if (!loopbackCertificate ||
        ![space.authenticationMethod isEqualToString:NSURLAuthenticationMethodServerTrust] ||
        ![space.host isEqualToString:@"localhost"] || space.port != 43179) {
        complete(NSURLSessionAuthChallengePerformDefaultHandling, nil);
        return;
    }

    SecTrustRef trust = space.serverTrust;
    SecCertificateRef leaf = trust ? SecTrustGetCertificateAtIndex(trust, 0) : NULL;
    CFDataRef data = leaf ? SecCertificateCopyData(leaf) : NULL;
    BOOL matches = data && [loopbackCertificate isEqualToData:(__bridge NSData *)data];
    if (data) CFRelease(data);

    if (matches) {
        complete(NSURLSessionAuthChallengeUseCredential, [NSURLCredential credentialForTrust:trust]);
    } else {
        complete(NSURLSessionAuthChallengeCancelAuthenticationChallenge, nil);
    }
}
@end
