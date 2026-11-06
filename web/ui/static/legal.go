package static

import (
	"context"

	. "maragu.dev/gomponents"
	. "maragu.dev/gomponents/html"
)

// PrivacyPolicy is the static privacy policy; hostURL is the site's canonical root URL.
func PrivacyPolicy(_ context.Context, hostURL string) Node {
	return Group{Raw(privacyHead), A(Href(hostURL), Text(hostURL)), Raw(privacyTail)}
}

// TermsOfService is the static terms of service page.
func TermsOfService(_ context.Context) Node {
	return Raw(termsOfService)
}

const privacyHead = `<div class="segment-panel reset-list">
    <h1>Privacy Policy</h1>
    <p><i>Last updated: 22 October 2024</i></p>

    <p>We're not the biggest fans of long legal documents, but we need this page.</p>

    <p>In a non-legally binding fashion, the Privacy Policy on Kilonova boils down to:</p>

    <ul>
        <li>We store what you submit on the website (submissions, biography, etc).</li>
        <li>We will protect personal information by reasonable security safeguards against loss or theft, as well as unauthorized access, disclosure, copying, use or modification.</li>
        <li>You can request to have your personal information deleted from the website.</li>
        <li>We will collect personal information by lawful and fair means and, where appropriate, with the knowledge or consent of the individual concerned.</li>
        <li>Personal data should be relevant to the purposes for which it is to be used, and, to the extent necessary for those purposes, should be accurate, complete, and up-to-date.</li>
        <li>We will make readily available to customers information about our policies and practices relating to the management of personal information.</li>
    </ul>
</div>

<div class="segment-panel reset-list">

<h1>Kilonova Privacy Policy</h1>

<p>This Privacy Policy describes how Kilonova (the "Site", "we", "us", or "our") collects, uses, and discloses your personal information when you visit, use our services, or make a purchase from `

const privacyTail = ` (the "Site") or otherwise communicate with us (collectively, the "Services"). For purposes of this Privacy Policy, "you" and "your" means you as the user of the Services, whether you are a customer, website visitor, or another individual whose information we have collected pursuant to this Privacy Policy.</p>

<p>Please read this Privacy Policy carefully. By using and accessing any of the Services, you agree to the collection, use, and disclosure of your information as described in this Privacy Policy. If you do not agree to this Privacy Policy, please do not use or access any of the Services.</p>

<h2>Changes to This Privacy Policy</h2>

<p>We may update this Privacy Policy from time to time, including to reflect changes to our practices or for other operational, legal, or regulatory reasons. We will post the revised Privacy Policy on the Site, update the "Last updated" date and take any other steps required by applicable law.</p>

<h2>How We Collect and Use Your Personal Information</h2>

<p>To provide the Services, we collect and have collected personal information about you from a variety of sources, as set out below. The information that we collect and use varies depending on how you interact with us.</p>

<p>In addition to the specific uses set out below, we may use information we collect about you to communicate with you, provide the Services, comply with any applicable legal obligations, enforce any applicable terms of service, and to protect or defend the Services, our rights, and the rights of our users or others.</p>

<h2>What Personal Information We Collect</h2>

<p>The types of personal information we obtain about you depends on how you interact with our Site and use our Services. When we use the term "personal information", we are referring to information that identifies, relates to, describes or can be associated with you. The following sections describe the categories and specific types of personal information we collect.</p>

<h2>Information We Collect Directly from You</h2>

<p>Information that you directly submit to us through our Services may include, but are not limited to:</p>

<ul>
    <li>Basic details including your name, email and preferred language of display.</li>
    <li>Account information including your username, password (hashed and secured), biography, etc.</li>
    <li>Solutions and other content (blog posts, comments) submitted to the website.</li>
    <li>Support information including the information you choose to include in communications with us, for example, when sending a message through the Services.</li>
</ul>

<p>Some features of the Services may require you to directly provide us with certain information about yourself. You may elect not to provide this information, but doing so may prevent you from using or accessing these features.</p>

<h2>Information We Collect through Cookies</h2>

<p>We also automatically collect certain information about your interaction with the Services ("Usage Data"). To do this, we may use cookies, pixels and similar technologies ("Cookies"). Usage Data may include information about how you access and use our Site and your account, including device information, browser information, information about your network connection, your IP address and other information regarding your interaction with the Services.</p>

<h2>Information We Obtain from Third Parties</h2>

<p>Finally, we may obtain information about you from third parties, including from vendors and service providers who may collect information on our behalf, such as:</p>

<ul>
    <li>Integrations such as Discord, BuyMeACoffee, Stripe or Paypal (payment processors).</li>
</ul>

<p>Any information we obtain from third parties will be treated in accordance with this Privacy Policy. We are not responsible or liable for the accuracy of the information provided to us by third parties and are not responsible for any third party's policies or practices. For more information, see the section below, Third Party Websites and Links.</p>

<h2>How We Use Your Personal Information</h2>

<ul>
    <li>Providing Products and Services. We use your personal information to provide you with the Services in order to perform our contract with you, including to process your payments, fulfill your orders, to send notifications to you related to you account, purchases, returns, exchanges or other transactions, to create, maintain and otherwise manage your account, to arrange for shipping, facilitate any returns and exchanges and to enable you to post reviews.</li>
    <li>Marketing and Advertising. We use your personal information for marketing and promotional purposes, such as to send marketing, advertising and promotional communications by email, text message or postal mail, and to show you advertisements for products or services. This may include using your personal information to better tailor the Services and advertising on our Site and other websites.</li>
    <li>Security and Fraud Prevention. We use your personal information to detect, investigate or take action regarding possible fraudulent, illegal or malicious activity. If you choose to use the Services and register an account, you are responsible for keeping your account credentials safe. We highly recommend that you do not share your username, password, or other access details with anyone else. If you believe your account has been compromised, please contact us immediately.</li>
    <li>Communicating with you. We use your personal information to provide you with customer support and improve our Services. This is in our legitimate interests in order to be responsive to you, to provide effective services to you, and to maintain our business relationship with you.</li>
</ul>

<h2>Cookies</h2>

<p>Like many websites, we use Cookies on our Site. We keep our usage of Cookies on the Site to a functional minimum (including to remember your preferences), to run analytics (in our legitimate interests to administer, improve and optimize the Services).</p>

<p>Most browsers automatically accept Cookies by default, but you can choose to set your browser to remove or reject Cookies through your browser controls. Please keep in mind that removing or blocking Cookies can negatively impact your user experience and may cause some of the Services, including certain features and general functionality, to work incorrectly or no longer be available.</p>

<h2>How We Disclose Personal Information</h2>

<p>In certain circumstances, we may disclose your personal information to third parties for legitimate purposes subject to this Privacy Policy. Such circumstances may include:</p>

<ul>
    <li>With vendors or other third parties who perform services on our behalf (e.g., IT management, payment processing, data analytics, customer support, cloud storage, fulfillment and shipping).</li>
    <li>When you direct, request us or otherwise consent to our disclosure of certain information to third parties, such as to ship you products or through your use of social media widgets or login integrations, with your consent.</li>
    <li>In order to comply with any applicable legal obligations (including to respond to subpoenas, search warrants and similar requests), to enforce any applicable terms of service, and to protect or defend the Services, our rights, and the rights of our users or others.</li>
</ul>

<h2>User Generated Content</h2>

<p>The Services may enable you to post user-generated content. If you choose to submit user generated content to any public area of the Services, this content can be public and accessible by anyone.</p>

<p>We do not control who will have access to the information that you choose to make available to others, and cannot ensure that parties who have access to such information will respect your privacy or keep it secure. We are not responsible for the privacy or security of any information that you make publicly available, or for the accuracy, use or misuse of any information that you disclose or receive from third parties.</p>

<h2>Third Party Websites and Links</h2>

<p>Our Site may provide links to websites or other online platforms operated by third parties. If you follow links to sites not affiliated or controlled by us, you should review their privacy and security policies and other terms and conditions. We do not guarantee and are not responsible for the privacy or security of such sites, including the accuracy, completeness, or reliability of information found on these sites. Information you provide on public or semi-public venues, including information you share on third-party social networking platforms may also be viewable by other users of the Services and/or users of those third-party platforms without limitation as to its use by us or by a third party. Our inclusion of such links does not, by itself, imply any endorsement of the content on such platforms or of their owners or operators, except as disclosed on the Services.</p>

<h2>Children's Data</h2>

<p>The Services are not intended to be used by children, and we do not knowingly collect any personal information about children. If you are the parent or guardian of a child who has provided us with their personal information, you may contact us using the contact details set out below to request that it be deleted.</p>

<h2>Security and Retention of Your Information</h2>

<p>Please be aware that no security measures are perfect or impenetrable, and we cannot guarantee "perfect security." In addition, any information you send to us may not be secure while in transit. We recommend that you do not use unsecure channels to communicate sensitive or confidential information to us.</p>

<p>How long we retain your personal information depends on different factors, such as whether we need the information to maintain your account, to provide the Services, comply with legal obligations, resolve disputes or enforce other applicable contracts and policies.</p>

<h2>Your Rights and Choices</h2>

<p>Depending on where you live, you may have some or all of the rights listed below in relation to your personal information. However, these rights are not absolute, may apply only in certain circumstances and, in certain cases, we may decline your request as permitted by law.</p>

<ul>
    <li>Right to Access / Know. You may have a right to request access to personal information that we hold about you, including details relating to the ways in which we use and share your information.</li>
    <li>Right to Delete. You may have a right to request that we delete personal information we maintain about you.</li>
    <li>Right to Correct. You may have a right to request that we correct inaccurate personal information we maintain about you.</li>
    <li>Right of Portability. You may have a right to receive a copy of the personal information we hold about you and to request that we transfer it to a third party, in certain circumstances and with certain exceptions.</li>
    <li>Withdrawal of Consent: Where we rely on consent to process your personal information, you may have the right to withdraw this consent.</li>
    <li>Appeal: You may have a right to appeal our decision if we decline to process your request. You can do so by replying directly to our denial.</li>
</ul>

<p>You may exercise any of these rights where indicated on our Site or by contacting us using the contact details provided below.</p>

<p>We will not discriminate against you for exercising any of these rights. We may need to collect information from you to verify your identity, such as your email address or account information, before providing a substantive response to the request. In accordance with applicable laws, You may designate an authorized agent to make requests on your behalf to exercise your rights. Before accepting such a request from an agent, we will require that the agent provide proof you have authorized them to act on your behalf, and we may need you to verify your identity directly with us. We will respond to your request in a timely manner as required under applicable law.</p>

<h2>Complaints</h2>

<p>If you have complaints about how we process your personal information, please contact us using the contact details provided below. If you are not satisfied with our response to your complaint, depending on where you live you may have the right to appeal our decision by contacting us using the contact details set out below, or lodge your complaint with your local data protection authority.</p>

International Users

<p>Please note that we may transfer, store and process your personal information outside the country you live in. If we transfer your personal information out of Europe, we will rely on recognized transfer mechanisms like the European Commission's Standard Contractual Clauses, or any equivalent contracts issued by the relevant competent authority of the UK, as relevant, unless the data transfer is to a country that has been determined to provide an adequate level of protection.</p>

<h2>Contact</h2>

<p>Should you have any questions about our privacy practices or this Privacy Policy, or if you would like to exercise any of the rights available to you, please email us using the address at the bottom.</p>

<p>For the purpose of applicable data protection laws, we are the data controller of your personal information. Our representatives are represented by the admins of this website.</p>

</div>`

const termsOfService = `<div class="segment-panel reset-list">
    <h1>Terms of Service</h1>
    <p><i>Last updated: 22 October 2024</i></p>

    <p>We're not the biggest fans of long legal documents, but we need this page.</p>

    <p>In a non-legally binding fashion, the Terms of Serivce on Kilonova boil down to:</p>

    <ul>
        <li>You agree to use the website properly and to maintain its proper function.</li>
        <li>We don't assume responsibility for the user-uploaded content on the website.</li>
        <li>We reserve the right to disqualify people from using the service if they don't fairly use it.</li>
        <li>Be civil. The admins and other people with permissions have the right to take down any content or accounts without warning.</li>
    </ul>
</div>

<div class="segment-panel reset-list">
<h1>Kilonova Terms of Service</h1>

<h2>Overview</h2>

<p> Throughout the site, the terms “we”, “us” and “our” refer to Kilonova and its staff. We offer this website, including all information, tools and Services available from this site to you, the user, conditioned upon your acceptance of all terms, conditions, policies and notices stated here.</p>

<p> By visiting our site and/ or purchasing something from us, you engage in our “Service” and agree to be bound by the following terms and conditions (“Terms of Service”, “Terms”), including those additional terms and conditions and policies referenced herein and/or available by hyperlink. These Terms of Service apply to all users of the site, including without limitation users who are browsers, vendors, customers, merchants, and/ or contributors of content.</p>

<p> Please read these Terms of Service carefully before accessing or using our website. By accessing or using any part of the site, you agree to be bound by these Terms of Service. If you do not agree to all the terms and conditions of this agreement, then you may not access the website or use any Services. If these Terms of Service are considered an offer, acceptance is expressly limited to these Terms of Service.</p>

<p> Any new features or tools which are added to the current store shall also be subject to the Terms of Service. You can review the most current version of the Terms of Service at any time on this page. We reserve the right to update, change or replace any part of these Terms of Service by posting updates and/or changes to our website. It is your responsibility to check this page periodically for changes. Your continued use of or access to the website following the posting of any changes constitutes acceptance of those changes. </p>

<h2>Usage terms</h2>

<p>By agreeing to these Terms of Service, you represent that you are at least the minimum legal age of your region to create an account (which is usually 16). Children may use an account established by their parent or legal guardian with the approval of such a person.</p>

<p>You may not use our products for any illegal or unauthorized purpose nor may you, in the use of the Service, violate any laws in your jurisdiction (including but not limited to copyright laws).</p>

<p>You must not transmit any worms or viruses or any code of a destructive nature or attempt to run malicious code on the Service.</p>

<p>A breach or violation of any of the Terms will result in an immediate termination of your Services.</p>

<h2>General Conditions</h2>

<p>We reserve the right to refuse service to anyone for any reason at any time.</p>

<p>You understand that your content, may be transferred unencrypted and involve (a) transmissions over various networks; and (b) changes to conform and adapt to technical requirements of connecting networks or devices.</p>

<p>You agree to respect the license of the content on the website and not reproduce, duplicate or exploit of all portions of the website.</p>

<p>The headings used in this agreement are included for convenience only and will not limit or otherwise affect these Terms.</p>

<h2>Accuracy, completeness and timeliness of information</h2>

<p>We are not responsible if information made available on this site is not accurate, complete or current. The material on this site is provided for general information only and should not be relied upon or used as the sole basis for making decisions without consulting primary, more accurate, more complete or more timely sources of information. Any reliance on the material on this site is at your own risk.</p>

<p>This site may contain certain historical information. Historical information, necessarily, is not current and is provided for your reference only. We reserve the right to modify the contents of this site at any time, but we have no obligation to update any information on our site. You agree that it is your responsibility to monitor changes to our site.</p>

<h2>Modifications to the service</h2>

<p>We reserve the right at any time to modify or discontinue the Service (or any part or content thereof) without notice at any time.</p>

<p>We shall not be liable to you or to any third-party for any modification, price change, suspension or discontinuance of the Service.</p>

<h2>Optional tools</h2>

<p>We may provide you with access to third-party tools over which we neither monitor nor have any control nor input.</p>

<p>You acknowledge and agree that we provide access to such tools ”as is” and “as available” without any warranties, representations or conditions of any kind and without any endorsement. We shall have no liability whatsoever arising from or relating to your use of optional third-party tools.</p>

<p>Any use by you of the optional tools offered through the site is entirely at your own risk and discretion and you should ensure that you are familiar with and approve of the terms on which tools are provided by the relevant third-party provider(s).</p>

<p>We may also, in the future, offer new Services and/or features through the website (including, the release of new tools and resources). Such new features and/or Services shall also be subject to these Terms of Service.</p>

<h2>Third-party links</h2>

<p>Certain content, products and Services available via our Service may include materials from third-parties.</p>

<p>Third-party links on this site may direct you to third-party websites that are not affiliated with us. We are not responsible for examining or evaluating the content or accuracy and we do not warrant and will not have any liability or responsibility for any third-party materials or websites, or for any other materials, products, or Services of third-parties.</p>

<p>We are not liable for any harm or damages related to the purchase or use of goods, Services, resources, content, or any other transactions made in connection with any third-party websites. Please review carefully the third-party's policies and practices and make sure you understand them before you engage in any transaction. Complaints, claims, concerns, or questions regarding third-party products should be directed to the third-party.</p>

<h2>User comments, feedback and other submissions</h2>

<p>We may, but have no obligation to, monitor, edit or remove content that we determine in our sole discretion to be unlawful, offensive, threatening, libelous, defamatory, pornographic, obscene or otherwise objectionable or violates any party’s intellectual property or these Terms of Service.</p>
<p>You agree that your comments will not violate any right of any third-party, including copyright, trademark, privacy, personality or other personal or proprietary right. You further agree that your comments will not contain libelous or otherwise unlawful, abusive or obscene material, or contain any computer virus or other malware that could in any way affect the operation of the Service or any related website. You may not use a false e-mail address, pretend to be someone other than yourself, or otherwise mislead us or third-parties as to the origin of any comments. You are solely responsible for any comments you make and their accuracy. We take no responsibility and assume no liability for any comments posted by you or any third-party.</p>

<h2>Personal Information</h2>

Your submission of personal information through the store is governed by our Privacy Policy. To view our Privacy Policy, please see <a href="/privacyPolicy">this page</a>.

<h2>Errors, inaccuracies and omissions</h2>

<p>Occasionally there may be information on our site or in the Service that contains typographical errors, inaccuracies or omissions. We reserve the right to correct any errors, inaccuracies or omissions, and to change or update information if any information in the Service or on any related website is inaccurate at any time without prior notice.</p>

<p>We undertake no obligation to update, amend or clarify information in the Service or on any related website. No specified update or refresh date applied in the Service or on any related website, should be taken to indicate that all information in the Service or on any related website has been modified or updated.</p>

<h2>Prohibited Uses</h2>

<p>In addition to other prohibitions as set forth in the Terms of Service, you are prohibited from using the site or its content:</p>
<p>(a) for any unlawful purpose; (b) to solicit others to perform or participate in any unlawful acts; (c) to violate any international, federal, provincial or state regulations, rules, laws, or local ordinances; (d) to infringe upon or violate our intellectual property rights or the intellectual property rights of others; (e) to harass, abuse, insult, harm, defame, slander, disparage, intimidate, or discriminate based on gender, sexual orientation, religion, ethnicity, race, age, national origin, or disability; (f) to submit false or misleading information;</p>
<p>(g) to upload or transmit viruses or any other type of malicious code that will or may be used in any way that will affect the functionality or operation of the Service or of any related website, other websites, or the Internet; (h) to collect or track the personal information of others; (i) to spam, phish, pharm, pretext, spider, crawl, or scrape; (j) for any obscene or immoral purpose; or (k) to interfere with or circumvent the security features of the Service or any related website, other websites, or the Internet. We reserve the right to terminate your use of the Service or any related website for violating any of the prohibited uses.</p>

<h2>Disclaimer of warranties; limitation of liability</h2>

<p>We do not guarantee, represent or warrant that your use of our service will be uninterrupted, timely, secure or error-free.</p>

<p>We do not warrant that the results that may be obtained from the use of the service will be accurate or reliable.</p>

<p>In the event that any provision of these Terms of Service is determined to be unlawful, void or unenforceable, such provision shall nonetheless be enforceable to the fullest extent permitted by applicable law, and the unenforceable portion shall be deemed to be severed from these Terms of Service, such determination shall not affect the validity and enforceability of any other remaining provisions.</p>

<p>The obligations and liabilities of the parties incurred prior to the termination date shall survive the termination of this agreement for all purposes.</p>

<p>These Terms of Service are effective unless and until terminated by either you or us. You may terminate these Terms of Service at any time by notifying us that you no longer wish to use our Services, or when you cease using our site.</p>
<p>If in our sole judgment you fail, or we suspect that you have failed, to comply with any term or provision of these Terms of Service, we also may terminate this agreement at any time without notice and you will remain liable for all amounts due up to and including the date of termination; and/or accordingly may deny you access to our Services (or any part thereof).</p>

<p>The failure of us to exercise or enforce any right or provision of these Terms of Service shall not constitute a waiver of such right or provision.</p>

<p>These Terms of Service and any policies or operating rules posted by us on this site or in respect to The Service constitutes the entire agreement and understanding between you and us and govern your use of the Service, superseding any prior or contemporaneous agreements, communications and proposals, whether oral or written, between you and us (including, but not limited to, any prior versions of the Terms of Service).</p>
<p>Any ambiguities in the interpretation of these Terms of Service shall not be construed against the drafting party.</p>

<p>These Terms of Service and any separate agreements whereby we provide you Services shall be governed by and construed in accordance with the laws of the European Union.</p>

<h2>Changes to terms of service</h2>

<p>You can review the most current version of the Terms of Service at any time at this page.</p>

<p>We reserve the right, at our sole discretion, to update, change or replace any part of these Terms of Service by posting updates and changes to our website. It is your responsibility to check our website periodically for changes. Your continued use of or access to our website or the Service following the posting of any changes to these Terms of Service constitutes acceptance of those changes.</p>

<h2>Contact</h2>

<p>Questions about the Terms of Service should be sent to us to the email in the footer of this page.</p>`
