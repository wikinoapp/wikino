# typed: strict
# frozen_string_literal: true

class Attachment < T::Struct
  extend T::Sig

  include T::Struct::ActsAsComparable

  const :id, T::Wikino::DatabaseId
  const :space_id, T::Wikino::DatabaseId
  const :filename, String
  const :content_type, String
  const :byte_size, Integer
  const :attached_space_member_id, T::Wikino::DatabaseId
  const :attached_at, ActiveSupport::TimeWithZone
  const :url, T.nilable(String)

  # ファイルサイズを人間が読みやすい形式で返す
  sig { returns(String) }
  def human_readable_size
    ActiveSupport::NumberHelper.number_to_human_size(byte_size)
  end

  # 画像ファイルかどうかを判定
  sig { returns(T::Boolean) }
  def image?
    content_type.start_with?("image/")
  end

  # サポートされているプレビュー可能な画像形式かどうか
  sig { returns(T::Boolean) }
  def previewable_image?
    %w[image/jpeg image/jpg image/png image/gif image/webp].include?(content_type)
  end
end
