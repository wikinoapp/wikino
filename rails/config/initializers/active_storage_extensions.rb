# typed: false
# frozen_string_literal: true

Rails.configuration.to_prepare do
  ActiveSupport.on_load(:active_storage_blob) do
    include BlobProcessable
  end
end

# ActiveStorage::AnalyzeJob downloads the blob to extract image metadata. When
# the underlying storage object is already gone — e.g. the attachment was purged
# before this enqueued job ran, or a direct upload never completed — the download
# raises ActiveStorage::FileNotFoundError. Such a blob can never be analyzed, so
# retrying or surfacing it as an error is pointless; discard the job quietly.
#
# This mirrors how Wikino already treats a missing storage object as benign in
# its own pipeline (BlobProcessable#download_to_tempfile rescues the same error,
# Attachments::ProcessService checks existence before processing) and follows the
# ApplicationJob guidance that jobs are safe to discard once their underlying
# records are no longer available. AnalyzeJob ships from the activestorage gem,
# so the rescue is wired here instead of in the job body.
#
# The constant is not yet autoloaded while initializers run, so the rescue is set
# inside to_prepare (the same hook used for the blob extension above). Because
# discard_on appends a handler every time it runs and to_prepare re-runs on each
# code reload, guard against registering the same handler more than once.
#
# [Ja] ActiveStorage::AnalyzeJob は画像メタデータ抽出のために blob をダウンロード
# する。実体オブジェクトが既に消えている場合 (このジョブの実行前に添付が purge された、
# ダイレクトアップロードが完了しなかった等) はダウンロードが
# ActiveStorage::FileNotFoundError を raise する。そうした blob は二度と analyze
# できないため、リトライやエラー報告は無意味であり、ジョブを静かに discard する。
#
# これは Wikino が独自パイプラインで実体欠落を既に benign 扱いしているのと同じ方針
# (BlobProcessable#download_to_tempfile は同じ例外を rescue し、
# Attachments::ProcessService は処理前に存在チェックする)。また「underlying records
# が無くなったジョブは discard してよい」という ApplicationJob の指針にも沿う。
# AnalyzeJob は activestorage gem 由来のため、ジョブ本体ではなくここで rescue を設定する。
#
# 定数は initializer 実行時点ではまだ autoload されていないため、rescue の設定は
# to_prepare 内で行う (上の blob 拡張と同じフック)。discard_on は呼ぶたびにハンドラを
# 追加し、to_prepare はコードリロードのたびに再実行されるため、同じハンドラを二重登録
# しないようガードする。
Rails.configuration.to_prepare do
  already_registered = ActiveStorage::AnalyzeJob.rescue_handlers.any? do |class_name, _handler|
    class_name == "ActiveStorage::FileNotFoundError"
  end

  ActiveStorage::AnalyzeJob.discard_on(ActiveStorage::FileNotFoundError) unless already_registered
end
