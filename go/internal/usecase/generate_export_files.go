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
	// exportContentTypeは、アーカイブを取得したときにストレージが返す値。ダウンロードの
	// リンクをたどったブラウザが、表示しようとせず保存するようにする。
	exportContentType = "application/zip"

	// exportAttachmentConcurrencyは同時に取得する添付ファイルの数。複製にかかる時間はバイト数
	// よりストレージとの往復に支配されるため、1つずつ取得すると添付ファイルの多いスペースが必要以上
	// に長くかかる。
	exportAttachmentConcurrency = 8

	// exportOutcomeTimeoutは、エクスポートがどう終わったかを記録する処理の上限。ジョブの
	// contextから切り離したcontextで動かすのは、最も記録する価値があるのが時間切れになった試行
	// だからである。そのときcontextは既にキャンセルされており、新しいcontextを用意しなければ
	// 失敗が記録も送信もされずに終わる。
	exportOutcomeTimeout = 30 * time.Second
)

// ExportSenderはエクスポートが送るメールを送信する。UseCaseがemailパッケージではなく
// これに依存することで、テンプレートのレンダリングがApplication層の外に留まる。nilでもよい。
// メールの設定が無いデプロイがこれにあたり、エクスポート自体は動いて、結果の通知だけが行われない。
type ExportSender interface {
	SendSucceeded(ctx context.Context, to, downloadURL, appURL, locale string) error
	SendFailed(ctx context.Context, to, exportURL, appURL, locale string) error
}

// GenerateExportFilesUsecaseはスペースをMarkdownファイルと添付ファイルのZIPとして
// 書き出し、アップロードし、依頼したメンバーに結果を伝える。
//
// 処理は意図的に冪等にしている。中断した試行は再開せず先頭からやり直し、同じスペースからは同じ
// オブジェクトキーに同じアーカイブができる。2つの試行が重ならないことを保証するのはエクスポート
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

// NewGenerateExportFilesUsecaseはGenerateExportFilesUsecaseを生成する。
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

// GenerateExportFilesInputは1つのエクスポートを生成するための入力パラメータ。
//
// FinalAttemptは、この試行の後にキューが再試行するかどうかを表す。失敗を記録するのは最後の試行の
// ときだけである。それより早く記録すると、リトライが抜け出せない終端の状態にエクスポートが残り、
// ストレージの一時的な失敗がエクスポートを終わらせてしまう。
type GenerateExportFilesInput struct {
	ExportID     model.ExportID
	SpaceID      model.SpaceID
	FinalAttempt bool
}

// Executeはエクスポートを生成し、どう終わったかを記録する。
func (uc *GenerateExportFilesUsecase) Execute(ctx context.Context, input GenerateExportFilesInput) error {
	export, err := uc.exportRepo.MarkStarted(ctx, input.ExportID, input.SpaceID)
	if err != nil {
		// この試行は、開始を記録できないまま諦めた。エクスポートはqueuedのままであり、
		// queuedのエクスポートはスペースを保つ。そのため最後の試行は、もう誰も動かさない状態
		// ではなく結果を残す必要がある。
		uc.recordFailure(ctx, input.ExportID, input.SpaceID, input.FinalAttempt, err)
		return fmt.Errorf("エクスポートの開始記録に失敗: %w", err)
	}
	if export == nil {
		// エクスポートは既に終端の状態にある。完了したジョブのリトライも、置き換えられた試行も
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

// generateはheartbeatを更新しながらアーカイブを生成・アップロードし、結果を記録・通知する。
// 成功への遷移前にheartbeatを止め、結果処理と後片付けにはそれぞれ期限を設ける。
func (uc *GenerateExportFilesUsecase) generate(ctx context.Context, export *model.Export) error {
	generationCtx, stopHeartbeat := uc.startHeartbeat(ctx, export)
	defer stopHeartbeat()

	objectKey, err := uc.buildAndUpload(generationCtx, export)
	stopHeartbeat()
	if err != nil {
		return err
	}

	// 結果処理は生成のキャンセルから切り離すが、時間の上限は設ける。
	// 自分の成功を置き換えと誤認しないよう、heartbeatを先に止める。
	ctx, cancel := context.WithTimeout(context.WithoutCancel(ctx), exportOutcomeTimeout)
	defer cancel()
	succeeded, err := uc.exportRepo.MarkSucceeded(ctx, export.ID, export.SpaceID, objectKey)
	if err != nil {
		return fmt.Errorf("エクスポートの成功記録に失敗: %w", err)
	}
	if succeeded == nil {
		// この試行は書き込み中に停止したと判断され、置き換えられている。そのアーカイブはもう
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

// buildAndUploadはアーカイブを一時ファイルへ書き出してアップロードし、保存先のキーを返す。
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

// planArchiveはスペースを読み、そのアーカイブの構成を決める。ページと、それらが参照する
// 添付ファイルはそれぞれ1クエリで読むため、行数そのもの以上にページ数へ比例するコストは無い。
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

// writeArchiveは添付ファイルを取得し、ZIPを書き出す。取得は先に並列で行い、ZIPの書き出しは
// 逐次の1パスであるため、取得したバイト列はその間だけ一時ディレクトリで待つ。
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

// writeArchiveAttachmentは取得済みの添付ファイル1つをアーカイブへ複製する。
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

// downloadAttachmentsはアーカイブのすべての添付ファイルをdirへ取得し、それぞれの場所を
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

// downloadAttachmentは添付ファイル1つを、アーカイブが期待する場所へ取得する。
func (uc *GenerateExportFilesUsecase) downloadAttachment(ctx context.Context, attachment *exportArchiveAttachment) error {
	body, err := uc.objectStorage.Get(ctx, attachment.blobKey)
	if err != nil {
		return fmt.Errorf("添付ファイル %sの取得に失敗: %w", attachment.fileName, err)
	}
	defer func() { _ = body.Close() }()

	file, err := os.Create(attachment.localPath)
	if err != nil {
		return fmt.Errorf("添付ファイルの一時ファイルの作成に失敗: %w", err)
	}
	defer func() { _ = file.Close() }()

	if _, err := io.Copy(file, body); err != nil {
		return fmt.Errorf("添付ファイル %sの書き込みに失敗: %w", attachment.fileName, err)
	}

	return nil
}

// startHeartbeatは、返した関数が呼ばれるまでエクスポートが生きていることを報告し、
// エクスポートが置き換えられていたと分かった場合にキャンセルされるcontextを返す。
//
// 遅いエクスポートと止まったエクスポートを分けるのがheartbeatである。ページの多いスペースは時間が
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

// recordFailureは試行が失敗したことを記録する。ただし最後の試行のときだけであり、あわせて
// エクスポートを依頼したメンバーに伝える。
//
// エクスポートをモデルではなくIDで受け取るのは、それを読み取る前に試行が失敗しうるためである。
// モデルを得るはずだった遷移そのものが、失敗した書き込みである場合がこれにあたる。メールの宛先は、
// MarkFailedが返す遷移後の行から解決する。
//
// ジョブのcontextから切り離したcontextで動かすのは、時間切れになった試行が自身のcontextを
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

// deleteReplacedExportsは、このエクスポートが置き換えるエクスポートをアーカイブごと削除し、
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
		// 成功記録に失敗した場合もアップロード先を復元する。レコードより先にオブジェクトを消し、
		// 削除に失敗してもZIPの追跡情報を残して再試行できるようにする。
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

// deleteObjectはアーカイブを削除できたか返す。失敗を記録し、呼び出し元がレコードを残せるようにする。
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

// notifyはエクスポートを開始したメンバーに、結果をメールで伝える。
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

// sendNotificationは宛先を解決し、エクスポートの結果に応じたメールを送る。スペースや
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

// exportObjectKeyはエクスポートのアーカイブの保存先。接頭辞によって、Rails版が同じバケットの
// ルートへ書いている添付ファイルとアーカイブが混ざらない。
func exportObjectKey(export *model.Export) string {
	return fmt.Sprintf("exports/%s/%s.zip", export.SpaceID, export.ID)
}

// exportURLはエクスポートの経過を追う画面のアドレス。
func exportURL(appURL string, spaceIdentifier model.SpaceIdentifier, exportID model.ExportID) string {
	return fmt.Sprintf("%s/s/%s/settings/exports/%s", appURL, spaceIdentifier, exportID)
}

// exportDownloadURLはエクスポートのアーカイブをダウンロードするアドレス。
func exportDownloadURL(appURL string, spaceIdentifier model.SpaceIdentifier, exportID model.ExportID) string {
	return exportURL(appURL, spaceIdentifier, exportID) + "/download"
}

// deleteLegacyObjectsは別レコードが使うblobを保持し、旧Rails ZIPを削除する。
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
