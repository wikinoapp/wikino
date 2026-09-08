package usecase

import (
	"archive/zip"
	"context"
	"errors"
	"fmt"
	"io"
	"log/slog"
	"os"
	"path/filepath"
	"strconv"
	"time"

	"golang.org/x/sync/errgroup"

	"github.com/wikinoapp/wikino/go/internal/config"
	"github.com/wikinoapp/wikino/go/internal/exportfile"
	"github.com/wikinoapp/wikino/go/internal/model"
	"github.com/wikinoapp/wikino/go/internal/repository"
	"github.com/wikinoapp/wikino/go/internal/storage"
)

const (
	// exportContentType is what the storage answers with when the archive is fetched, so that a
	// browser following the download link saves it instead of trying to render it.
	//
	// [Ja] exportContentType は、アーカイブを取得したときにストレージが返す値。ダウンロードの
	// リンクをたどったブラウザが、表示しようとせず保存するようにする。
	exportContentType = "application/zip"

	// exportAttachmentConcurrency is how many attachments are fetched at once. The time it takes to
	// copy them is dominated by the round trip to the storage rather than by the bytes, so fetching
	// one at a time makes a space with many attachments take far longer than it has to.
	//
	// [Ja] exportAttachmentConcurrency は同時に取得する添付ファイルの数。複製にかかる時間はバイト数
	// よりストレージとの往復に支配されるため、1 つずつ取得すると添付ファイルの多いスペースが必要以上
	// に長くかかる。
	exportAttachmentConcurrency = 8

	// exportOutcomeTimeout bounds the work that records how an export ended. It runs on a context
	// detached from the job's, because the case worth recording most is the attempt that ran out of
	// time: its context is already cancelled by then, and without a fresh one the failure would go
	// unrecorded and unsent.
	//
	// [Ja] exportOutcomeTimeout は、エクスポートがどう終わったかを記録する処理の上限。ジョブの
	// context から切り離した context で動かすのは、最も記録する価値があるのが時間切れになった試行
	// だからである。そのとき context は既にキャンセルされており、新しい context を用意しなければ
	// 失敗が記録も送信もされずに終わる。
	exportOutcomeTimeout = 30 * time.Second
)

// ExportSender sends the mails an export produces. The UseCase depends on this rather than on the
// email package, so that rendering a template stays out of the Application layer. It may be nil,
// which is what a deployment with no mail configured gets: the export still runs, and only the
// announcement of how it ended is left out.
//
// [Ja] ExportSender はエクスポートが送るメールを送信する。UseCase が email パッケージではなく
// これに依存することで、テンプレートのレンダリングが Application 層の外に留まる。nil でもよい。
// メールの設定が無いデプロイがこれにあたり、エクスポート自体は動いて、結果の通知だけが行われない。
type ExportSender interface {
	SendSucceeded(ctx context.Context, to, downloadURL, appURL, locale string) error
	SendFailed(ctx context.Context, to, exportURL, appURL, locale string) error
}

// GenerateExportFilesUsecase writes a space out as a ZIP of Markdown files and attachments,
// uploads it, and tells the member who asked for it how it went.
//
// The work is idempotent on purpose: an interrupted attempt is not resumed but started over, and
// the same space gives the same archive under the same object key. What keeps two attempts from
// overlapping is the status of the export itself, which every transition here is conditional on.
//
// [Ja] GenerateExportFilesUsecase はスペースを Markdown ファイルと添付ファイルの ZIP として
// 書き出し、アップロードし、依頼したメンバーに結果を伝える。
//
// 処理は意図的に冪等にしている。中断した試行は再開せず先頭からやり直し、同じスペースからは同じ
// オブジェクトキーに同じアーカイブができる。2 つの試行が重ならないことを保証するのはエクスポート
// 自身の状態で、ここでの遷移はすべてそれを条件とする。
type GenerateExportFilesUsecase struct {
	cfg             *config.Config
	exportRepo      *repository.ExportRepository
	spaceRepo       *repository.SpaceRepository
	spaceMemberRepo *repository.SpaceMemberRepository
	userRepo        *repository.UserRepository
	topicRepo       *repository.TopicRepository
	pageRepo        *repository.PageRepository
	attachmentRepo  *repository.AttachmentRepository
	objectStorage   storage.ObjectStorage
	sender          ExportSender
}

// NewGenerateExportFilesUsecase creates a GenerateExportFilesUsecase.
//
// [Ja] NewGenerateExportFilesUsecase は GenerateExportFilesUsecase を生成する。
func NewGenerateExportFilesUsecase(
	cfg *config.Config,
	exportRepo *repository.ExportRepository,
	spaceRepo *repository.SpaceRepository,
	spaceMemberRepo *repository.SpaceMemberRepository,
	userRepo *repository.UserRepository,
	topicRepo *repository.TopicRepository,
	pageRepo *repository.PageRepository,
	attachmentRepo *repository.AttachmentRepository,
	objectStorage storage.ObjectStorage,
	sender ExportSender,
) *GenerateExportFilesUsecase {
	return &GenerateExportFilesUsecase{
		cfg:             cfg,
		exportRepo:      exportRepo,
		spaceRepo:       spaceRepo,
		spaceMemberRepo: spaceMemberRepo,
		userRepo:        userRepo,
		topicRepo:       topicRepo,
		pageRepo:        pageRepo,
		attachmentRepo:  attachmentRepo,
		objectStorage:   objectStorage,
		sender:          sender,
	}
}

// GenerateExportFilesInput holds what it takes to generate one export.
//
// FinalAttempt says whether the queue will try again after this attempt. A failure is only
// recorded on the last one: recording it earlier would leave the export in a terminal status that
// the retry cannot move out of, so a momentary failure of the storage would end the export for
// good.
//
// [Ja] GenerateExportFilesInput は 1 つのエクスポートを生成するための入力パラメータ。
//
// FinalAttempt は、この試行の後にキューが再試行するかどうかを表す。失敗を記録するのは最後の試行の
// ときだけである。それより早く記録すると、リトライが抜け出せない終端の状態にエクスポートが残り、
// ストレージの一時的な失敗がエクスポートを終わらせてしまう。
type GenerateExportFilesInput struct {
	ExportID     model.ExportID
	SpaceID      model.SpaceID
	FinalAttempt bool
}

// Execute generates the export and records how it ended.
//
// [Ja] Execute はエクスポートを生成し、どう終わったかを記録する。
func (uc *GenerateExportFilesUsecase) Execute(ctx context.Context, input GenerateExportFilesInput) error {
	export, err := uc.exportRepo.MarkStarted(ctx, input.ExportID, input.SpaceID)
	if err != nil {
		// The attempt gave up before it could record that it started. The export is still
		// queued, and a queued export holds its space, so the last attempt has to leave a
		// result behind rather than a status nothing will ever move.
		//
		// [Ja] この試行は、開始を記録できないまま諦めた。エクスポートは queued のままであり、
		// queued のエクスポートはスペースを保つ。そのため最後の試行は、もう誰も動かさない状態
		// ではなく結果を残す必要がある。
		uc.recordFailure(ctx, input.ExportID, input.SpaceID, input.FinalAttempt, err)
		return fmt.Errorf("エクスポートの開始記録に失敗: %w", err)
	}
	if export == nil {
		// The export already reached a terminal status, which a retry of a finished job or a
		// replaced attempt both look like. There is nothing left to generate.
		//
		// [Ja] エクスポートは既に終端の状態にある。完了したジョブのリトライも、置き換えられた試行も
		// この形になる。生成すべきものはもう残っていない。
		slog.InfoContext(ctx, "エクスポートは既に完了しているため生成を行いません",
			"export_id", input.ExportID.String(),
		)
		return nil
	}

	if err := uc.generate(ctx, export); err != nil {
		uc.recordFailure(ctx, export.ID, export.SpaceID, input.FinalAttempt, err)
		return err
	}

	return nil
}

// generate builds and uploads the archive with a heartbeat, then records and announces the outcome.
// Heartbeat ends before the success transition; outcome and cleanup each have bounded contexts.
//
// [Ja] generate は heartbeat を更新しながらアーカイブを生成・アップロードし、結果を記録・通知する。
// 成功への遷移前に heartbeat を止め、結果処理と後片付けにはそれぞれ期限を設ける。
func (uc *GenerateExportFilesUsecase) generate(ctx context.Context, export *model.Export) error {
	generationCtx, stopHeartbeat := uc.startHeartbeat(ctx, export)
	defer stopHeartbeat()

	objectKey, err := uc.buildAndUpload(generationCtx, export)
	stopHeartbeat()
	if err != nil {
		return err
	}

	// Outcome processing must outlive generation cancellation, but remain bounded.
	// Stop heartbeat first so it cannot mistake our own success for replacement.
	//
	// [Ja] 結果処理は生成のキャンセルから切り離すが、時間の上限は設ける。
	// 自分の成功を置き換えと誤認しないよう、heartbeat を先に止める。
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), exportOutcomeTimeout)
	defer cancel()
	succeeded, err := uc.exportRepo.MarkSucceeded(ctx, export.ID, export.SpaceID, objectKey)
	if err != nil {
		return fmt.Errorf("エクスポートの成功記録に失敗: %w", err)
	}
	if succeeded == nil {
		// This attempt was declared stopped and replaced while it was writing. Its archive belongs
		// to no export any more, so it is removed instead of left to sit in the bucket.
		//
		// [Ja] この試行は書き込み中に停止したと判断され、置き換えられている。そのアーカイブはもう
		// どのエクスポートのものでもないため、バケットに残さず削除する。
		uc.deleteObject(ctx, objectKey)
		return nil
	}

	uc.notify(ctx, succeeded, true)
	cleanupCtx, cleanupCancel := context.WithTimeout(context.WithoutCancel(ctx), exportOutcomeTimeout)
	defer cleanupCancel()
	uc.deleteReplacedExports(cleanupCtx, succeeded)

	return nil
}

// buildAndUpload writes the archive to a temporary file and uploads it, and returns the key it was
// stored under. The archive goes through a file rather than memory because it carries every
// attachment of the space, whose total size is not known before it is read.
//
// The key is derived from the export, so an attempt that starts over writes over what an earlier
// attempt left behind instead of adding an orphan to the bucket.
//
// [Ja] buildAndUpload はアーカイブを一時ファイルへ書き出してアップロードし、保存先のキーを返す。
// メモリではなくファイルを経由するのは、アーカイブがスペースのすべての添付ファイルを含み、その
// 合計サイズが読み終えるまで分からないためである。
//
// キーはエクスポートから決まるため、やり直した試行は前の試行が残したものを上書きする。バケットに
// 迷子のオブジェクトを増やすことはない。
func (uc *GenerateExportFilesUsecase) buildAndUpload(ctx context.Context, export *model.Export) (string, error) {
	archive, err := uc.planArchive(ctx, export.SpaceID)
	if err != nil {
		return "", err
	}

	file, err := os.CreateTemp("", "wikino-export-*.zip")
	if err != nil {
		return "", fmt.Errorf("エクスポートの一時ファイルの作成に失敗: %w", err)
	}
	defer func() {
		_ = file.Close()
		if err := os.Remove(file.Name()); err != nil && !errors.Is(err, os.ErrNotExist) {
			slog.WarnContext(ctx, "エクスポートの一時ファイルの削除に失敗しました", "path", file.Name(), "error", err)
		}
	}()

	if err := uc.writeArchive(ctx, archive, file); err != nil {
		return "", err
	}

	if _, err := file.Seek(0, io.SeekStart); err != nil {
		return "", fmt.Errorf("エクスポートの一時ファイルの読み直しに失敗: %w", err)
	}

	objectKey := exportObjectKey(export)
	if err := uc.objectStorage.Upload(ctx, storage.UploadInput{
		Key:         objectKey,
		Body:        file,
		ContentType: exportContentType,
	}); err != nil {
		return "", fmt.Errorf("エクスポートのアップロードに失敗: %w", err)
	}

	return objectKey, nil
}

// planArchive reads the space and decides the layout of its archive. The pages and the attachments
// they reference are each read in one query, so the cost does not grow with the number of pages
// beyond the rows themselves.
//
// [Ja] planArchive はスペースを読み、そのアーカイブの構成を決める。ページと、それらが参照する
// 添付ファイルはそれぞれ 1 クエリで読むため、行数そのもの以上にページ数へ比例するコストは無い。
func (uc *GenerateExportFilesUsecase) planArchive(ctx context.Context, spaceID model.SpaceID) (*exportArchive, error) {
	topics, err := uc.topicRepo.ListActiveBySpace(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("トピック一覧の取得に失敗: %w", err)
	}

	pages, err := uc.pageRepo.ListActiveBySpace(ctx, spaceID)
	if err != nil {
		return nil, fmt.Errorf("ページ一覧の取得に失敗: %w", err)
	}

	pageIDs := make([]model.PageID, len(pages))
	for i, page := range pages {
		pageIDs[i] = page.ID
	}

	pageAttachments, err := uc.attachmentRepo.ListByPageIDsAndSpace(ctx, pageIDs, spaceID)
	if err != nil {
		return nil, fmt.Errorf("添付ファイル一覧の取得に失敗: %w", err)
	}

	return planExportArchive(topics, pages, pageAttachments), nil
}

// writeArchive fetches the attachments and writes the ZIP. The fetching happens first and in
// parallel, while writing the ZIP is one sequential pass, so the fetched bytes wait in a temporary
// directory in between.
//
// [Ja] writeArchive は添付ファイルを取得し、ZIP を書き出す。取得は先に並列で行い、ZIP の書き出しは
// 逐次の 1 パスであるため、取得したバイト列はその間だけ一時ディレクトリで待つ。
func (uc *GenerateExportFilesUsecase) writeArchive(ctx context.Context, archive *exportArchive, out io.Writer) error {
	dir, err := os.MkdirTemp("", "wikino-export-attachments-")
	if err != nil {
		return fmt.Errorf("添付ファイルの一時ディレクトリの作成に失敗: %w", err)
	}
	defer func() {
		if err := os.RemoveAll(dir); err != nil {
			slog.WarnContext(ctx, "添付ファイルの一時ディレクトリの削除に失敗しました", "path", dir, "error", err)
		}
	}()

	if err := uc.downloadAttachments(ctx, archive, dir); err != nil {
		return err
	}

	writer := zip.NewWriter(out)
	for _, topic := range archive.topics {
		for _, page := range topic.pages {
			entry, err := writer.Create(topic.dir + "/" + page.fileName)
			if err != nil {
				return fmt.Errorf("ページのエントリの作成に失敗: %w", err)
			}
			if _, err := io.WriteString(entry, page.body); err != nil {
				return fmt.Errorf("ページのエントリの書き込みに失敗: %w", err)
			}
		}

		for _, attachment := range topic.attachments {
			if err := writeArchiveAttachment(writer, topic, attachment); err != nil {
				return err
			}
		}
	}

	if err := writer.Close(); err != nil {
		return fmt.Errorf("エクスポートのZIPの書き出しに失敗: %w", err)
	}

	return nil
}

// writeArchiveAttachment copies one fetched attachment into the archive.
//
// [Ja] writeArchiveAttachment は取得済みの添付ファイル 1 つをアーカイブへ複製する。
func writeArchiveAttachment(writer *zip.Writer, topic *exportArchiveTopic, attachment *exportArchiveAttachment) error {
	entry, err := writer.Create(topic.dir + "/" + exportfile.AttachmentDirName + "/" + attachment.fileName)
	if err != nil {
		return fmt.Errorf("添付ファイルのエントリの作成に失敗: %w", err)
	}

	file, err := os.Open(attachment.localPath)
	if err != nil {
		return fmt.Errorf("取得済みの添付ファイルを開けませんでした: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(entry, file); err != nil {
		return fmt.Errorf("添付ファイルのエントリの書き込みに失敗: %w", err)
	}

	return nil
}

// downloadAttachments fetches every attachment of the archive into dir and records where each one
// landed.
//
// A fetch that fails ends the whole export. An archive missing an attachment would look complete
// to whoever downloads it, and they would have no way of telling that something is not there.
//
// [Ja] downloadAttachments はアーカイブのすべての添付ファイルを dir へ取得し、それぞれの場所を
// 記録する。
//
// 取得に失敗したらエクスポート全体を終わらせる。添付ファイルの欠けたアーカイブは、ダウンロードした
// 人には完全なものに見え、何かが欠けていることを知る手立てが無いためである。
func (uc *GenerateExportFilesUsecase) downloadAttachments(ctx context.Context, archive *exportArchive, dir string) error {
	group, ctx := errgroup.WithContext(ctx)
	group.SetLimit(exportAttachmentConcurrency)

	index := 0
	for _, topic := range archive.topics {
		for _, attachment := range topic.attachments {
			attachment.localPath = filepath.Join(dir, strconv.Itoa(index))
			index++

			group.Go(func() error {
				return uc.downloadAttachment(ctx, attachment)
			})
		}
	}

	return group.Wait()
}

// downloadAttachment fetches one attachment into the place the archive expects it.
//
// [Ja] downloadAttachment は添付ファイル 1 つを、アーカイブが期待する場所へ取得する。
func (uc *GenerateExportFilesUsecase) downloadAttachment(ctx context.Context, attachment *exportArchiveAttachment) error {
	body, err := uc.objectStorage.Get(ctx, attachment.blobKey)
	if err != nil {
		return fmt.Errorf("添付ファイル %s の取得に失敗: %w", attachment.fileName, err)
	}
	defer func() { _ = body.Close() }()

	file, err := os.Create(attachment.localPath)
	if err != nil {
		return fmt.Errorf("添付ファイルの一時ファイルの作成に失敗: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, body); err != nil {
		return fmt.Errorf("添付ファイル %s の書き込みに失敗: %w", attachment.fileName, err)
	}

	return nil
}

// startHeartbeat reports that the export is alive until the returned function is called, and
// returns a context that is cancelled if the export turns out to have been replaced.
//
// The heartbeat is what separates a slow export from a stopped one: a space with many pages can
// take a while, and time alone would say nothing about whether anyone is still working on it.
//
// [Ja] startHeartbeat は、返した関数が呼ばれるまでエクスポートが生きていることを報告し、
// エクスポートが置き換えられていたと分かった場合にキャンセルされる context を返す。
//
// 遅いエクスポートと止まったエクスポートを分けるのが heartbeat である。ページの多いスペースは時間が
// かかることがあり、経過時間だけでは誰かがまだ作業しているかどうかを何も語らない。
func (uc *GenerateExportFilesUsecase) startHeartbeat(ctx context.Context, export *model.Export) (context.Context, func()) {
	ctx, cancel := context.WithCancel(ctx)
	done := make(chan struct{})

	go func() {
		defer close(done)

		ticker := time.NewTicker(model.ExportHeartbeatInterval)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				alive, err := uc.exportRepo.UpdateHeartbeat(ctx, export.ID, export.SpaceID)
				if err != nil {
					slog.WarnContext(ctx, "エクスポートのheartbeatの更新に失敗しました",
						"export_id", export.ID.String(),
						"error", err,
					)
					continue
				}
				if !alive {
					slog.WarnContext(ctx, "エクスポートが置き換えられたため生成を打ち切ります",
						"export_id", export.ID.String(),
					)
					cancel()
					return
				}
			}
		}
	}()

	return ctx, func() {
		cancel()
		<-done
	}
}

// recordFailure records that the attempt failed, but only on the last one, and tells the member
// who asked for the export.
//
// It takes the export by ID rather than as a model because an attempt can fail before it has read
// one: the transition that would have produced it is the write that failed. MarkFailed returns the
// row it moved, which is what the mail is addressed from.
//
// It runs on a context detached from the job's, because an attempt that ran out of time reaches
// here with its own context already cancelled, and that is precisely the failure worth telling
// someone about.
//
// [Ja] recordFailure は試行が失敗したことを記録する。ただし最後の試行のときだけであり、あわせて
// エクスポートを依頼したメンバーに伝える。
//
// エクスポートをモデルではなく ID で受け取るのは、それを読み取る前に試行が失敗しうるためである。
// モデルを得るはずだった遷移そのものが、失敗した書き込みである場合がこれにあたる。メールの宛先は、
// MarkFailed が返す遷移後の行から解決する。
//
// ジョブの context から切り離した context で動かすのは、時間切れになった試行が自身の context を
// キャンセルされた状態でここへ来るためである。そしてそれこそが、誰かに伝える価値のある失敗である。
func (uc *GenerateExportFilesUsecase) recordFailure(ctx context.Context, exportID model.ExportID, spaceID model.SpaceID, finalAttempt bool, cause error) {
	if !finalAttempt {
		slog.WarnContext(ctx, "エクスポートの生成に失敗しました。再試行されます",
			"export_id", exportID.String(),
			"error", cause,
		)
		return
	}

	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), exportOutcomeTimeout)
	defer cancel()

	slog.ErrorContext(ctx, "エクスポートの生成に失敗しました",
		"export_id", exportID.String(),
		"error", cause,
	)

	failed, err := uc.exportRepo.MarkFailed(ctx, exportID, spaceID)
	if err != nil {
		slog.ErrorContext(ctx, "エクスポートの失敗記録に失敗しました",
			"export_id", exportID.String(),
			"error", err,
		)
		return
	}
	if failed == nil {
		return
	}

	uc.notify(ctx, failed, false)
}

// deleteReplacedExports removes the exports this one replaces, together with their archives, so
// that a space keeps only its latest successful export.
//
// A failure here is logged and no more: the archive is written and the member has been told about
// it, and the next successful export tries the same cleanup again.
//
// [Ja] deleteReplacedExports は、このエクスポートが置き換えるエクスポートをアーカイブごと削除し、
// スペースが最新の成功したエクスポートだけを保つようにする。
//
// ここでの失敗はログに記録するだけにする。アーカイブは書き出され、メンバーにも伝わっており、次に
// 成功したエクスポートが同じ後片付けを改めて試みるためである。
func (uc *GenerateExportFilesUsecase) deleteReplacedExports(ctx context.Context, current *model.Export) {
	replaced, err := uc.exportRepo.ListOlderBySpace(ctx, current.SpaceID, current.ID)
	if err != nil {
		slog.ErrorContext(ctx, "置き換えられたエクスポートの取得に失敗しました",
			"export_id", current.ID.String(),
			"error", err,
		)
		return
	}

	for _, export := range replaced {
		// Recover the upload destination even if recording success failed. Delete the object before
		// its record so failed cleanup can retry without losing track of the ZIP.
		//
		// [Ja] 成功記録に失敗した場合もアップロード先を復元する。レコードより先にオブジェクトを消し、
		// 削除に失敗しても ZIP の追跡情報を残して再試行できるようにする。
		objectKey := exportObjectKey(export)
		if export.ObjectKey != nil {
			objectKey = *export.ObjectKey
		}
		if !uc.deleteObject(ctx, objectKey) {
			continue
		}
		if !uc.deleteLegacyObjects(ctx, export) {
			continue
		}

		if err := uc.exportRepo.Delete(ctx, export.ID, current.SpaceID); err != nil {
			slog.ErrorContext(ctx, "置き換えられたエクスポートの削除に失敗しました",
				"export_id", export.ID.String(),
				"error", err,
			)
		}
	}
}

// deleteObject reports whether an archive was removed, logging failures so callers can retain its record.
//
// [Ja] deleteObject はアーカイブを削除できたか返す。失敗を記録し、呼び出し元がレコードを残せるようにする。
func (uc *GenerateExportFilesUsecase) deleteObject(ctx context.Context, objectKey string) bool {
	if err := uc.objectStorage.Delete(ctx, objectKey); err != nil {
		slog.ErrorContext(ctx, "エクスポートのアーカイブの削除に失敗しました",
			"object_key", objectKey,
			"error", err,
		)
		return false
	}
	return true
}

// notify mails the member who started the export how it went.
//
// A mail that cannot be sent is logged and not retried through the job: the export has already
// reached a terminal status, so a retry would find nothing left to do and the mail would never go
// out anyway.
//
// [Ja] notify はエクスポートを開始したメンバーに、結果をメールで伝える。
//
// 送れなかったメールはログに記録し、ジョブのリトライには載せない。エクスポートは既に終端の状態へ
// 到達しており、リトライしても行うことは残っておらず、どのみちメールは送られないためである。
func (uc *GenerateExportFilesUsecase) notify(ctx context.Context, export *model.Export, succeeded bool) {
	if err := uc.sendNotification(ctx, export, succeeded); err != nil {
		slog.ErrorContext(ctx, "エクスポートの結果メールの送信に失敗しました",
			"export_id", export.ID.String(),
			"error", err,
		)
	}
}

// sendNotification resolves the recipient and sends the mail for how the export ended. A space or a
// member that is gone leaves nobody to tell, and a deployment with no mail configured has no way to
// tell them; neither is an error.
//
// [Ja] sendNotification は宛先を解決し、エクスポートの結果に応じたメールを送る。スペースや
// メンバーが失われている場合は伝える相手が居ないだけであり、メールの設定が無いデプロイには伝える
// 手段が無いだけである。どちらもエラーではない。
func (uc *GenerateExportFilesUsecase) sendNotification(ctx context.Context, export *model.Export, succeeded bool) error {
	if uc.sender == nil {
		return nil
	}

	space, err := uc.spaceRepo.FindByID(ctx, export.SpaceID)
	if err != nil {
		return fmt.Errorf("スペースの取得に失敗: %w", err)
	}
	if space == nil {
		return nil
	}

	spaceMembers, err := uc.spaceMemberRepo.FindByIDs(ctx, []model.SpaceMemberID{export.QueuedByID}, export.SpaceID)
	if err != nil {
		return fmt.Errorf("スペースメンバーの取得に失敗: %w", err)
	}
	if len(spaceMembers) == 0 {
		return nil
	}

	user, err := uc.userRepo.FindByID(ctx, spaceMembers[0].UserID)
	if err != nil {
		return fmt.Errorf("ユーザーの取得に失敗: %w", err)
	}
	if user == nil || user.Email == "" {
		return nil
	}

	appURL := uc.cfg.AppURL()
	if succeeded {
		return uc.sender.SendSucceeded(ctx, user.Email, exportDownloadURL(appURL, space.Identifier, export.ID), appURL, user.Locale.Code())
	}

	return uc.sender.SendFailed(ctx, user.Email, exportURL(appURL, space.Identifier, export.ID), appURL, user.Locale.Code())
}

// exportObjectKey is where the archive of an export is stored. The prefix keeps the archives apart
// from the attachments, which the Rails version writes to the root of the same bucket.
//
// [Ja] exportObjectKey はエクスポートのアーカイブの保存先。接頭辞によって、Rails 版が同じバケットの
// ルートへ書いている添付ファイルとアーカイブが混ざらない。
func exportObjectKey(export *model.Export) string {
	return fmt.Sprintf("exports/%s/%s.zip", export.SpaceID, export.ID)
}

// exportURL is the address of the screen an export is followed on.
//
// [Ja] exportURL はエクスポートの経過を追う画面のアドレス。
func exportURL(appURL string, spaceIdentifier model.SpaceIdentifier, exportID model.ExportID) string {
	return fmt.Sprintf("%s/s/%s/settings/exports/%s", appURL, spaceIdentifier, exportID)
}

// exportDownloadURL is the address the archive of an export is downloaded from.
//
// [Ja] exportDownloadURL はエクスポートのアーカイブをダウンロードするアドレス。
func exportDownloadURL(appURL string, spaceIdentifier model.SpaceIdentifier, exportID model.ExportID) string {
	return exportURL(appURL, spaceIdentifier, exportID) + "/download"
}

// deleteLegacyObjects removes old Rails ZIPs while preserving blobs used by other records.
// Metadata is retained until every object deletion succeeds, allowing a later cleanup to retry.
//
// [Ja] deleteLegacyObjects は別レコードが使う blob を保持し、旧 Rails ZIP を削除する。
// 次の後片付けで再試行できるよう、全オブジェクトを削除するまでメタデータを保持する。
func (uc *GenerateExportFilesUsecase) deleteLegacyObjects(ctx context.Context, export *model.Export) bool {
	files, err := uc.exportRepo.ListLegacyFiles(ctx, export.ID, export.SpaceID)
	if err != nil {
		slog.ErrorContext(ctx, "旧エクスポートのファイル取得に失敗しました", "export_id", export.ID.String(), "error", err)
		return false
	}
	for _, file := range files {
		if !file.Shared && !uc.deleteObject(ctx, file.Key) {
			return false
		}
	}
	return true
}
