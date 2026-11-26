//go:build bundle && cgo

#import <UIKit/UIKit.h>
#import <WebKit/WebKit.h>
#import <arpa/inet.h>

static NSInteger previewPort(NSURL *url) {
    if (url.port) return url.port.integerValue;
    return [url.scheme.lowercaseString isEqualToString:@"https"] ? 443 : 80;
}

static UIColor *previewColor(id value) {
    if (![value isKindOfClass:NSString.class] || [value length] != 7 ||
        ![value hasPrefix:@"#"]) return nil;
    unsigned int rgb;
    NSScanner *scanner = [NSScanner scannerWithString:[value substringFromIndex:1]];
    if (![scanner scanHexInt:&rgb] || !scanner.isAtEnd) return nil;
    return [UIColor colorWithRed:((rgb >> 16) & 255) / 255.0
                          green:((rgb >> 8) & 255) / 255.0
                           blue:(rgb & 255) / 255.0 alpha:1];
}

@interface BetterMusicLivePreview : UIViewController <WKNavigationDelegate, UITabBarDelegate>
@property (nonatomic, strong) NSURL *previewURL;
@property (nonatomic, strong) WKWebView *webView;
@property (nonatomic, strong) UITabBar *tabBar;
- (void)selectTabFromPage:(WKScriptMessage *)message;
@end

@interface BetterMusicLivePreviewMessages : NSObject <WKScriptMessageHandler>
@property (nonatomic, weak) BetterMusicLivePreview *preview;
@end

@implementation BetterMusicLivePreviewMessages
- (void)userContentController:(WKUserContentController *)controller
      didReceiveScriptMessage:(WKScriptMessage *)message {
    [self.preview selectTabFromPage:message];
}
@end

@implementation BetterMusicLivePreview
- (void)viewDidLoad {
    [super viewDidLoad];
    self.title = @"Live redesign preview";
    self.view.backgroundColor = UIColor.systemBackgroundColor;
    self.navigationItem.leftBarButtonItem =
        [[UIBarButtonItem alloc] initWithBarButtonSystemItem:UIBarButtonSystemItemDone
                                                    target:self action:@selector(close)];
    self.navigationItem.rightBarButtonItem =
        [[UIBarButtonItem alloc] initWithBarButtonSystemItem:UIBarButtonSystemItemRefresh
                                                    target:self action:@selector(loadPreview)];

    WKWebViewConfiguration *configuration = [[WKWebViewConfiguration alloc] init];
    configuration.websiteDataStore = WKWebsiteDataStore.nonPersistentDataStore;
    BetterMusicLivePreviewMessages *messages = [[BetterMusicLivePreviewMessages alloc] init];
    messages.preview = self;
    [configuration.userContentController addScriptMessageHandler:messages name:@"betterMusicLivePreviewTabs"];
    [configuration.userContentController addScriptMessageHandler:messages name:@"betterMusicLivePreviewDock"];
    self.webView = [[WKWebView alloc] initWithFrame:CGRectZero configuration:configuration];
    self.webView.navigationDelegate = self;
    self.webView.translatesAutoresizingMaskIntoConstraints = NO;

    self.tabBar = [[UITabBar alloc] init];
    self.tabBar.translatesAutoresizingMaskIntoConstraints = NO;
    self.tabBar.delegate = self;
    self.tabBar.items = @[
        [[UITabBarItem alloc] initWithTitle:@"Artists" image:[UIImage systemImageNamed:@"person.2"] tag:0],
        [[UITabBarItem alloc] initWithTitle:@"Album detail" image:[UIImage systemImageNamed:@"opticaldisc"] tag:1]
    ];
    self.tabBar.selectedItem = self.tabBar.items.firstObject;
    [self.view addSubview:self.webView];
    [self.view addSubview:self.tabBar];
    UILayoutGuide *safeArea = self.view.safeAreaLayoutGuide;
    [NSLayoutConstraint activateConstraints:@[
        [self.webView.topAnchor constraintEqualToAnchor:safeArea.topAnchor],
        [self.webView.leadingAnchor constraintEqualToAnchor:safeArea.leadingAnchor],
        [self.webView.trailingAnchor constraintEqualToAnchor:safeArea.trailingAnchor],
        [self.webView.bottomAnchor constraintEqualToAnchor:self.tabBar.topAnchor],
        [self.tabBar.leadingAnchor constraintEqualToAnchor:self.view.leadingAnchor],
        [self.tabBar.trailingAnchor constraintEqualToAnchor:self.view.trailingAnchor],
        [self.tabBar.bottomAnchor constraintEqualToAnchor:safeArea.bottomAnchor],
        [self.tabBar.heightAnchor constraintEqualToConstant:49],
    ]];
    [self loadPreview];
}

- (void)loadPreview {
    self.tabBar.selectedItem = self.tabBar.items.firstObject;
    [self.webView loadRequest:[NSURLRequest requestWithURL:self.previewURL
                                             cachePolicy:NSURLRequestReloadIgnoringLocalCacheData
                                         timeoutInterval:15]];
}

- (void)close {
    [self dismissViewControllerAnimated:YES completion:nil];
}

- (BOOL)allowsURL:(NSURL *)url {
    return [url.scheme.lowercaseString isEqualToString:self.previewURL.scheme.lowercaseString] &&
        [url.host.lowercaseString isEqualToString:self.previewURL.host.lowercaseString] &&
        previewPort(url) == previewPort(self.previewURL) &&
        !url.user && !url.password &&
        [url.path isEqualToString:self.previewURL.path];
}

- (void)webView:(WKWebView *)webView decidePolicyForNavigationAction:(WKNavigationAction *)action
    decisionHandler:(void (^)(WKNavigationActionPolicy))decide {
    decide(action.targetFrame.isMainFrame && [self allowsURL:action.request.URL]
        ? WKNavigationActionPolicyAllow : WKNavigationActionPolicyCancel);
}

- (void)webView:(WKWebView *)webView decidePolicyForNavigationResponse:(WKNavigationResponse *)response
    decisionHandler:(void (^)(WKNavigationResponsePolicy))decide {
    decide(response.isForMainFrame && [self allowsURL:response.response.URL]
        ? WKNavigationResponsePolicyAllow : WKNavigationResponsePolicyCancel);
}

- (void)webView:(WKWebView *)webView didFailProvisionalNavigation:(WKNavigation *)navigation
    withError:(NSError *)error {
    if (error.code == NSURLErrorCancelled || !self.view.window || self.presentedViewController) return;
    UIAlertController *alert = [UIAlertController alertControllerWithTitle:@"Couldn’t load the live preview"
        message:error.localizedDescription
        preferredStyle:UIAlertControllerStyleAlert];
    [alert addAction:[UIAlertAction actionWithTitle:@"OK" style:UIAlertActionStyleDefault handler:nil]];
    [self presentViewController:alert animated:YES completion:nil];
}

- (void)selectTabFromPage:(WKScriptMessage *)message {
    if (!message.frameInfo.isMainFrame || ![self allowsURL:message.frameInfo.request.URL]) return;

    if ([message.name isEqualToString:@"betterMusicLivePreviewDock"]) {
        if (![message.body isKindOfClass:NSDictionary.class]) return;
        NSDictionary *config = message.body;
        NSArray *tabs = config[@"items"];
        if (![tabs isKindOfClass:NSArray.class] || tabs.count < 1 || tabs.count > 5) return;
        NSMutableArray<UITabBarItem *> *items = [NSMutableArray array];
        for (id entry in tabs) {
            if (![entry isKindOfClass:NSDictionary.class]) return;
            NSString *title = entry[@"title"];
            NSString *symbol = entry[@"symbol"];
            if (![title isKindOfClass:NSString.class] || ![symbol isKindOfClass:NSString.class]) return;
            [items addObject:[[UITabBarItem alloc] initWithTitle:title
                image:[UIImage systemImageNamed:symbol] tag:items.count]];
        }
        NSInteger selected = [self.tabBar.items indexOfObject:self.tabBar.selectedItem];
        self.tabBar.items = items;
        self.tabBar.selectedItem = items[selected >= 0 && selected < (NSInteger)items.count ? selected : 0];
        UITabBarAppearance *appearance = [[UITabBarAppearance alloc] init];
        [appearance configureWithDefaultBackground];
        UIColor *background = previewColor(config[@"background"]);
        UIColor *tint = previewColor(config[@"tint"]);
        UIColor *muted = previewColor(config[@"muted"]);
        if (background) appearance.backgroundColor = background;
        for (UITabBarItemAppearance *item in @[appearance.stackedLayoutAppearance,
             appearance.inlineLayoutAppearance, appearance.compactInlineLayoutAppearance]) {
            if (tint) {
                item.selected.iconColor = tint;
                item.selected.titleTextAttributes = @{NSForegroundColorAttributeName:tint};
            }
            if (muted) {
                item.normal.iconColor = muted;
                item.normal.titleTextAttributes = @{NSForegroundColorAttributeName:muted};
            }
        }
        self.tabBar.standardAppearance = appearance;
        self.tabBar.scrollEdgeAppearance = appearance;
        return;
    }
    if (![message.body isKindOfClass:NSNumber.class]) return;

    NSInteger index = [message.body integerValue];
    if (index >= 0 && index < (NSInteger)self.tabBar.items.count) {
        self.tabBar.selectedItem = self.tabBar.items[index];
    }
}

- (void)tabBar:(UITabBar *)tabBar didSelectItem:(UITabBarItem *)item {
    NSString *script = [NSString stringWithFormat:
        @"window.dispatchEvent(new CustomEvent('nativeTabSelected',{detail:{index:%ld}}));", (long)item.tag];
    [self.webView evaluateJavaScript:script completionHandler:nil];
}
@end

void openBetterMusicLivePreview(UIViewController *presenter, NSString *address) {
    NSURLComponents *url = [NSURLComponents componentsWithString:address];
    BOOL tailscaleHTTPS = [url.scheme.lowercaseString isEqualToString:@"https"] &&
        [url.host.lowercaseString hasSuffix:@".ts.net"] && (!url.port || url.port.integerValue == 443);
    struct in_addr addressIPv4;
    BOOL tailscaleIP = [url.scheme.lowercaseString isEqualToString:@"http"] && url.host &&
        inet_pton(AF_INET, url.host.UTF8String, &addressIPv4) == 1 &&
        (ntohl(addressIPv4.s_addr) & 0xffc00000) == 0x64400000 &&
        (!url.port || (url.port.integerValue > 0 && url.port.integerValue <= 65535));
    if ((!tailscaleHTTPS && !tailscaleIP) || url.user || url.password ||
        ![url.path isEqualToString:@"/preview/redesign"] || url.query || url.fragment) return;

    NSURL *previewURL = url.URL;
    if (!previewURL || !presenter.view.window || presenter.presentedViewController) return;
    BetterMusicLivePreview *preview = [[BetterMusicLivePreview alloc] init];
    preview.previewURL = previewURL;
    UINavigationController *navigation = [[UINavigationController alloc] initWithRootViewController:preview];
    navigation.modalPresentationStyle = UIModalPresentationFullScreen;
    [presenter presentViewController:navigation animated:YES completion:nil];
}
