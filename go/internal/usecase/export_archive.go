package usecase

import (
	"path"
	"strings"

	"github.com/wikinoapp/wikino/go/internal/exportfile"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
)

// exportPageExtはページを書き出すファイルの拡張子。アーカイブはObsidianのvaultとして
// 開けることを意図しており、ObsidianはノートをMarkdownのファイルから読む。
const exportPageExt = ".md"

// exportArchiveは1つのアーカイブの構成である。各トピックがどのディレクトリを、各ページと
// 各添付ファイルがどのファイルを持つのか、そして各ページのファイルが持つ本文を表す。
//
// 1バイトも書き出す前にすべてを決めるのは、ページの本文が他のページを最終的な名前で指すためで
// ある。その名前は、すべてのトピックとページが置換と重複の解決を経て初めて確定する。
type exportArchive struct {
	topics []*exportArchiveTopic
}

// exportArchiveTopicはアーカイブのトピック1つ分のディレクトリ。
type exportArchiveTopic struct {
	dir         string
	pages       []*exportArchivePage
	attachments []*exportArchiveAttachment
}

// exportArchivePageはアーカイブのMarkdownファイル1つ分で、その本文は書き換え済みである。
type exportArchivePage struct {
	fileName string
	body     string
}

// exportArchiveAttachmentは複製する添付ファイル1つ分。blobKeyは取得するオブジェクトを
// 指し、localPathは取得したバイト列がアーカイブへ書き出されるまで待つ場所である。
type exportArchiveAttachment struct {
	blobKey   string
	fileName  string
	localPath string
}

// plannedExportTopicは、名前が決まったトピックについて、構成決めの2巡目が必要とするものを
// 保持する。トピック自身、アーカイブでの並び順のページ、そして添付ファイルに与えられた名前である。
type plannedExportTopic struct {
	topic           *model.Topic
	archiveTopic    *exportArchiveTopic
	pages           []*model.Page
	attachmentNames map[model.AttachmentID]string
}

// planExportArchiveは、アーカイブの構成をその元になるスペースからすべて決める。
//
// アクティブなページを持たないトピックにはディレクトリを与えない。空のディレクトリはスペースに
// ついて何も語らず、Rails版も同じくそれを含めていなかった。
func planExportArchive(topics []*model.Topic, pages []*model.Page, pageAttachments []*repository.PageAttachment) *exportArchive {
	pagesByTopic := groupPagesByTopic(pages)
	attachmentsByPage := groupAttachmentsByPage(pageAttachments)

	archive := &exportArchive{}
	topicDeduper := exportfile.NewDeduper()

	// あるページのWikiリンクはどのトピックのページでも指せるため、本文を書き換えるより先に
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

// rewriteExportBodyはページの本文を、アーカイブが持つ形に変える。添付ファイルへの参照と
// Wikiリンクは隣にあるファイルを指し、ファイル名がもう綴っていないタイトルはfrontmatterに残る。
//
// どちらの書き換えも自身が走査した本文の位置を読むため、他方が拾う記法を書き込まない限り順に
// 適用して問題ない。実際どちらも書き込まない。添付ファイルのリンク先はパーセントエンコードされた
// 形になり、書き換えたWikiリンクは添付ファイルのパスを名乗らない。
func rewriteExportBody(page *model.Page, topicName string, pagePaths map[exportfile.PageRef]string, attachmentNames map[model.AttachmentID]string) string {
	body := exportfile.RewriteAttachmentLinks(page.Body, attachmentNames)
	body = exportfile.RewriteWikilinks(body, topicName, pagePaths)
	return exportfile.WithFrontmatter(body, pageTitle(page))
}

// planTopicAttachmentsはトピックのディレクトリの添付ファイルを埋め、それぞれがどの名前で
// 複製されるのかを返す。そのトピックの本文にある参照は、この名前を指す形へ書き換えられる。
//
// トピック内の複数のページから参照されている添付ファイルの複製は1つで、並び順はそれを最初に
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

// groupPagesByTopicはスペースのページをトピックごとにまとめる。渡された順序は保つ。
func groupPagesByTopic(pages []*model.Page) map[model.TopicID][]*model.Page {
	byTopic := map[model.TopicID][]*model.Page{}
	for _, page := range pages {
		byTopic[page.TopicID] = append(byTopic[page.TopicID], page)
	}
	return byTopic
}

// groupAttachmentsByPageはスペースの添付ファイルを、それを参照しているページごとにまとめる。
func groupAttachmentsByPage(pageAttachments []*repository.PageAttachment) map[model.PageID][]*model.Attachment {
	byPage := map[model.PageID][]*model.Attachment{}
	for _, pageAttachment := range pageAttachments {
		byPage[pageAttachment.PageID] = append(byPage[pageAttachment.PageID], pageAttachment.Attachment)
	}
	return byPage
}

// pageTitleはページのタイトルを文字列として返す。エクスポートの対象は公開済みのページだけ
// で、それらは必ずタイトルを持つため、空のタイトルは本来存在しないはずのレコードを意味する。
// exportfileはそれを名前の無いエントリにはせず、代替名へ置き換える。
func pageTitle(page *model.Page) string {
	if page.Title == nil {
		return ""
	}
	return *page.Title
}
