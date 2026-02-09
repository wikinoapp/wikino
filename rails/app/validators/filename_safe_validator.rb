# typed: strict
# frozen_string_literal: true

class FilenameSafeValidator < ActiveModel::EachValidator
  extend T::Sig

  # ファイル名として使用できない文字を含むかどうかをチェックする正規表現
  INVALID_CHARS_PATTERN = %r{[/\\:*?"<>|]}
  WINDOWS_RESERVED_NAMES = %w[
    CON PRN AUX NUL COM1 COM2 COM3 COM4 COM5 COM6 COM7 COM8 COM9
    LPT1 LPT2 LPT3 LPT4 LPT5 LPT6 LPT7 LPT8 LPT9
  ]

  sig { params(record: T.untyped, attribute: Symbol, value: T.nilable(String)).void }
  def validate_each(record, attribute, value)
    return if value.nil?

    if value.match?(INVALID_CHARS_PATTERN)
      record.errors.add(attribute, :contains_invalid_characters_html)
    end

    # 先頭・末尾のスペースとピリオドのチェック
    if value.start_with?(" ", ".") || value.end_with?(" ", ".")
      record.errors.add(attribute, :cannot_start_or_end_with_space_or_dot)
    end

    # Windowsの予約語チェック
    if WINDOWS_RESERVED_NAMES.include?(value.upcase)
      record.errors.add(attribute, :reserved)
    end
  end
end
