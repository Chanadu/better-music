//go:build bundle && cgo

#import <UIKit/UIKit.h>
#import <WebKit/WebKit.h>
#import <Security/Security.h>
#import <objc/runtime.h>

// Wails v3.0.0 uses this controller as its WKNavigationDelegate.
@interface WailsViewController : UIViewController <WKNavigationDelegate>
@property (nonatomic, strong) UITabBar *tabBar;
- (void)enableNativeTabs:(BOOL)enabled;
@end

extern void ios_native_tabs_set_items_json(const char *json);
void openBetterMusicLivePreview(UIViewController *presenter, NSString *address);

@interface BetterMusicPreviewTabs : NSObject <WKScriptMessageHandler>
@property (nonatomic, weak) WailsViewController *controller;
@end

@implementation BetterMusicPreviewTabs
- (void)userContentController:(WKUserContentController *)controller
      didReceiveScriptMessage:(WKScriptMessage *)message {
    WKSecurityOrigin *origin = message.frameInfo.securityOrigin;
    if (!message.frameInfo.isMainFrame ||
        ![origin.protocol isEqualToString:@"https"] ||
        ![origin.host isEqualToString:@"localhost"] || origin.port != 43179) {
        return;
    }

    if ([message.name isEqualToString:@"betterMusicLivePreview"]) {
        if ([message.webView.URL.path isEqualToString:@"/more"] &&
            [message.body isKindOfClass:[NSString class]]) {
            openBetterMusicLivePreview(self.controller, message.body);
        }
        return;
    }
    if (![message.body isKindOfClass:[NSNumber class]]) return;

    NSInteger index = [message.body integerValue];
    BOOL show = [message.webView.URL.path isEqualToString:@"/preview/redesign"] && (index == 0 || index == 1);
    WailsViewController *viewController = self.controller;
    [viewController enableNativeTabs:show];
    if (show) {
        // Wails queues the tab bar layout on the main thread. Update its selection
        // after it creates the bar, without sending a second tab selection event.
        dispatch_async(dispatch_get_main_queue(), ^{
            UITabBar *bar = viewController.tabBar;
            if (index < bar.items.count) bar.selectedItem = bar.items[index];
        });
    }
}
@end

static char previewTabsKey;

static void installPreviewTabs(WailsViewController *controller, WKWebView *webView) {
    if (objc_getAssociatedObject(webView, &previewTabsKey)) return;

    BetterMusicPreviewTabs *handler = [[BetterMusicPreviewTabs alloc] init];
    handler.controller = controller;
    WKUserContentController *content = webView.configuration.userContentController;
    [content addScriptMessageHandler:handler name:@"betterMusicPreviewTabs"];
    [content addScriptMessageHandler:handler name:@"betterMusicLivePreview"];
    // Full-page navigation out of the preview hides the native bar before the
    // next page renders, even if the next page has no Svelte navigation code.
    WKUserScript *reset = [[WKUserScript alloc] initWithSource:
        @"if (location.pathname !== '/preview/redesign') window.webkit.messageHandlers.betterMusicPreviewTabs.postMessage(-1);"
        injectionTime:WKUserScriptInjectionTimeAtDocumentEnd forMainFrameOnly:YES];
    [content addUserScript:reset];
    objc_setAssociatedObject(webView, &previewTabsKey, @YES, OBJC_ASSOCIATION_RETAIN_NONATOMIC);
}

static NSData *loopbackCertificate;

void pinBetterMusicLoopbackCertificate(const void *der, unsigned long length) {
    loopbackCertificate = [[NSData alloc] initWithBytes:der length:length];
    ios_native_tabs_set_items_json(
        "[{\"Title\":\"Artists\",\"SystemImage\":\"person.2\"},"
        "{\"Title\":\"Album detail\",\"SystemImage\":\"opticaldisc\"}]");
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
        installPreviewTabs(self, webView);
        complete(NSURLSessionAuthChallengeUseCredential, [NSURLCredential credentialForTrust:trust]);
    } else {
        complete(NSURLSessionAuthChallengeCancelAuthenticationChallenge, nil);
    }
}
@end
