package usecase

import (
	"path"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/exportfile"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// exportPageExt is the extension of the file a page is written as. The archive is meant to open as
// an Obsidian vault, which reads a note from a Markdown file.
//
// [Ja] exportPageExt はページを書き出すファイルの拡張子。アーカイブは Obsidian の vault として
// 開けることを意図しており、Obsidian はノートを Markdown のファイルから読む。
const exportPageExt = ".md"

// exportArchive is the layout of one archive: which directory each topic gets, which file each
// page and each attachment gets, and the text each page file holds.
//
// Everything is decided before a byte is written, because a page body points at other pages by
// the names they end up with, and those names are only settled once every topic and page has been
// through replacement and deduplication.
//
// [Ja] exportArchive は 1 つのアーカイブの構成である。各トピックがどのディレクトリを、各ページと
// 各添付ファイルがどのファイルを持つのか、そして各ページのファイルが持つ本文を表す。
//
// 1 バイトも書き出す前にすべてを決めるのは、ページの本文が他のページを最終的な名前で指すためで
// ある。その名前は、すべてのトピックとページが置換と重複の解決を経て初めて確定する。
type exportArchive struct {
	topics []*exportArchiveTopic
}

// exportArchiveTopic is one topic directory of the archive.
//
// [Ja] exportArchiveTopic はアーカイブのトピック 1 つ分のディレクトリ。
type exportArchiveTopic struct {
	dir         string
	pages       []*exportArchivePage
	attachments []*exportArchiveAttachment
}

// exportArchivePage is one Markdown file of the archive, with the text it holds already rewritten.
//
// [Ja] exportArchivePage はアーカイブの Markdown ファイル 1 つ分で、その本文は書き換え済みである。
type exportArchivePage struct {
	fileName string
	body     string
}

// exportArchiveAttachment is one copied attachment. blobKey names the object to fetch, and
// localPath is where the fetched bytes wait until the archive is written.
//
// [Ja] exportArchiveAttachment は複製する添付ファイル 1 つ分。blobKey は取得するオブジェクトを
// 指し、localPath は取得したバイト列がアーカイブへ書き出されるまで待つ場所である。
type exportArchiveAttachment struct {
	blobKey   string
	fileName  string
	localPath string
}

// plannedExportTopic carries what the second pass of the planning needs about a topic whose names
// are already decided: the topic itself, its pages in archive order, and the names its attachments
// were given.
//
// [Ja] plannedExportTopic は、名前が決まったトピックについて、構成決めの 2 巡目が必要とするものを
// 保持する。トピック自身、アーカイブでの並び順のページ、そして添付ファイルに与えられた名前である。
type plannedExportTopic struct {
	topic           *model.Topic
	archiveTopic    *exportArchiveTopic
	pages           []*model.Page
	attachmentNames map[model.AttachmentID]string
}

// planExportArchive decides the whole layout of the archive from the space it is made of.
//
// A topic with no active page gets no directory: an empty directory says nothing about the space,
// and the Rails version left it out as well.
//
// [Ja] planExportArchive は、アーカイブの構成をその元になるスペースからすべて決める。
//
// アクティブなページを持たないトピックにはディレクトリを与えない。空のディレクトリはスペースに
// ついて何も語らず、Rails 版も同じくそれを含めていなかった。
func planExportArchive(topics []*model.Topic, pages []*model.Page, pageAttachments []*repository.PageAttachment) *exportArchive {
	pagesByTopic := groupPagesByTopic(pages)
	attachmentsByPage := groupAttachmentsByPage(pageAttachments)

	archive := &exportArchive{}
	topicDeduper := exportfile.NewDeduper()

	// A wiki link of one page can name a page of any topic, so the whole table of page paths has to
	// exist before a body is rewritten. Naming everything comes first, and rewriting second.
	//
	// [Ja] あるページの Wiki リンクはどのトピックのページでも指せるため、本文を書き換えるより先に
	// ページのパスの対応表が揃っている必要がある。まずすべての名前を決め、書き換えはその後になる。
	pagePaths := map[exportfile.PageRef]string{}
	planned := make([]plannedExportTopic, 0, len(topics))

	for _, topic := range topics {
		topicPages := pagesByTopic[topic.ID]
		if len(topicPages) == 0 {
			continue
		}

		archiveTopic := &exportArchiveTopic{dir: topicDeduper.Unique(topic.Name, "")}
		pageDeduper := exportfile.NewDeduper()

		for _, page := range topicPages {
			title := pageTitle(page)
			fileName := pageDeduper.Unique(title, exportPageExt)

			archiveTopic.pages = append(archiveTopic.pages, &exportArchivePage{fileName: fileName})
			pagePaths[exportfile.PageRef{TopicName: topic.Name, PageTitle: title}] = archiveTopic.dir + "/" + fileName
		}

		attachmentNames := planTopicAttachments(archiveTopic, topicPages, attachmentsByPage)

		archive.topics = append(archive.topics, archiveTopic)
		planned = append(planned, plannedExportTopic{
			topic:           topic,
			archiveTopic:    archiveTopic,
			pages:           topicPages,
			attachmentNames: attachmentNames,
		})
	}

	for _, plan := range planned {
		for i, page := range plan.pages {
			plan.archiveTopic.pages[i].body = rewriteExportBody(page, plan.topic.Name, pagePaths, plan.attachmentNames)
		}
	}

	return archive
}

// rewriteExportBody turns the body of a page into what the archive carries: its attachment
// references and wiki links point at the files beside it, and its title survives in frontmatter
// even though the file name no longer spells it out.
//
// Both rewrites read positions of a body they scanned themselves, so running one after the other
// is safe as long as neither writes syntax the other picks up. Neither does: an attachment
// destination comes out percent encoded, and a rewritten wiki link names no attachment path.
//
// [Ja] rewriteExportBody はページの本文を、アーカイブが持つ形に変える。添付ファイルへの参照と
// Wiki リンクは隣にあるファイルを指し、ファイル名がもう綴っていないタイトルは frontmatter に残る。
//
// どちらの書き換えも自身が走査した本文の位置を読むため、他方が拾う記法を書き込まない限り順に
// 適用して問題ない。実際どちらも書き込まない。添付ファイルのリンク先はパーセントエンコードされた
// 形になり、書き換えた Wiki リンクは添付ファイルのパスを名乗らない。
func rewriteExportBody(page *model.Page, topicName string, pagePaths map[exportfile.PageRef]string, attachmentNames map[model.AttachmentID]string) string {
	body := exportfile.RewriteAttachmentLinks(page.Body, attachmentNames)
	body = exportfile.RewriteWikilinks(body, topicName, pagePaths)
	return exportfile.WithFrontmatter(body, pageTitle(page))
}

// planTopicAttachments fills in the attachments of a topic directory and returns the name each one
// is copied under, which is what the references in the bodies of that topic are rewritten to.
//
// An attachment referenced from several pages of the topic is copied once, and the first page that
// references it decides where it sits in the order.
//
// [Ja] planTopicAttachments はトピックのディレクトリの添付ファイルを埋め、それぞれがどの名前で
// 複製されるのかを返す。そのトピックの本文にある参照は、この名前を指す形へ書き換えられる。
//
// トピック内の複数のページから参照されている添付ファイルの複製は 1 つで、並び順はそれを最初に
// 参照しているページが決める。
func planTopicAttachments(archiveTopic *exportArchiveTopic, topicPages []*model.Page, attachmentsByPage map[model.PageID][]*model.Attachment) map[model.AttachmentID]string {
	deduper := exportfile.NewDeduper()
	names := map[model.AttachmentID]string{}

	for _, page := range topicPages {
		for _, attachment := range attachmentsByPage[page.ID] {
			if _, taken := names[attachment.ID]; taken {
				continue
			}

			ext := path.Ext(attachment.Filename)
			fileName := deduper.Unique(strings.TrimSuffix(attachment.Filename, ext), ext)

			names[attachment.ID] = fileName
			archiveTopic.attachments = append(archiveTopic.attachments, &exportArchiveAttachment{
				blobKey:  attachment.BlobKey,
				fileName: fileName,
			})
		}
	}

	return names
}

// groupPagesByTopic groups the pages of a space by their topic, keeping the order they came in.
//
// [Ja] groupPagesByTopic はスペースのページをトピックごとにまとめる。渡された順序は保つ。
func groupPagesByTopic(pages []*model.Page) map[model.TopicID][]*model.Page {
	byTopic := map[model.TopicID][]*model.Page{}
	for _, page := range pages {
		byTopic[page.TopicID] = append(byTopic[page.TopicID], page)
	}
	return byTopic
}

// groupAttachmentsByPage groups the attachments of a space by the page that references them.
//
// [Ja] groupAttachmentsByPage はスペースの添付ファイルを、それを参照しているページごとにまとめる。
func groupAttachmentsByPage(pageAttachments []*repository.PageAttachment) map[model.PageID][]*model.Attachment {
	byPage := map[model.PageID][]*model.Attachment{}
	for _, pageAttachment := range pageAttachments {
		byPage[pageAttachment.PageID] = append(byPage[pageAttachment.PageID], pageAttachment.Attachment)
	}
	return byPage
}

// pageTitle returns the title of a page as a string. Only published pages are exported and those
// always carry one, so an empty title means a record that should not exist; exportfile turns it
// into the substitute name rather than an entry with no name.
//
// [Ja] pageTitle はページのタイトルを文字列として返す。エクスポートの対象は公開済みのページだけ
// で、それらは必ずタイトルを持つため、空のタイトルは本来存在しないはずのレコードを意味する。
// exportfile はそれを名前の無いエントリにはせず、代替名へ置き換える。
func pageTitle(page *model.Page) string {
	if page.Title == nil {
		return ""
	}
	return *page.Title
}
