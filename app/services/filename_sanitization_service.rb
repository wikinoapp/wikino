# typed: strict
# frozen_string_literal: true

class FilenameSanitizationService
  extend T::Sig

  # ファイル名で使用できない文字
  INVALID_CHARACTERS = T.let(
    /[<>:"|?*\x00-\x1f]/,
    Regexp
  )

  # ファイル名の最大長
  MAX_FILENAME_LENGTH = T.let(255, Integer)

  # ファイル名の最大長（拡張子を除く）
  MAX_BASENAME_LENGTH = T.let(200, Integer)

  sig { params(filename: String).returns(String) }
  def self.sanitize(filename)
    new.sanitize(filename)
  end

  sig { params(filename: String).returns(String) }
  def sanitize(filename)
    # 空白文字をトリム
    sanitized = filename.strip

    # 無効な文字をアンダースコアに置換
    sanitized = sanitized.gsub(INVALID_CHARACTERS, "_")

    # パストラバーサルを防ぐためにパス区切り文字をアンダースコアに置換
    sanitized = sanitized.gsub(/[\/\\]/, "_")

    # 連続するアンダースコアを1つに縮める
    sanitized = sanitized.squeeze("_")

    # ファイル名がピリオドで始まる場合は削除（隠しファイルを防ぐ）
    sanitized = sanitized.sub(/^\.*/, "")

    # ファイル名と拡張子を分離
    basename = File.basename(sanitized, ".*")
    extension = File.extname(sanitized)

    # ベース名が空の場合はデフォルト名を使用
    basename = "file" if basename.empty?

    # ベース名の長さを制限
    if basename.length > MAX_BASENAME_LENGTH
      basename = basename[0...MAX_BASENAME_LENGTH]
    end

    # ファイル名を再構築
    result = "#{basename}#{extension}"

    # 全体の長さがMAX_FILENAME_LENGTHを超える場合は切り詰め
    if result.length > MAX_FILENAME_LENGTH
      # 拡張子を保持しつつ、ファイル名を切り詰め
      max_basename = MAX_FILENAME_LENGTH - extension.length
      if max_basename > 0
        basename = basename.not_nil![0...max_basename]
        result = "#{basename}#{extension}"
      else
        # 拡張子が長すぎる場合はファイル名全体を切り詰め
        result = result[0...MAX_FILENAME_LENGTH]
      end
    end

    # Windowsの予約名をチェック
    result = sanitize_reserved_names(result.not_nil!)

    result.not_nil!
  end

  private

  # Windowsの予約名をチェックして必要に応じて修正
  sig { params(filename: String).returns(String) }
  private def sanitize_reserved_names(filename)
    reserved_names = %w[
      CON PRN AUX NUL COM1 COM2 COM3 COM4 COM5 COM6 COM7 COM8 COM9
      LPT1 LPT2 LPT3 LPT4 LPT5 LPT6 LPT7 LPT8 LPT9
    ]

    basename = File.basename(filename, ".*")
    extension = File.extname(filename)

    # ベース名が予約名に一致する場合は修正
    if reserved_names.include?(basename.upcase)
      basename = "#{basename}_file"
    end

    "#{basename}#{extension}"
  end
end