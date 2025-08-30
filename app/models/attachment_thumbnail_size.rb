# typed: strict
# frozen_string_literal: true

class AttachmentThumbnailSize < T::Enum
  enums do
    Card = new("card") # ページ一覧のカード用
    Og = new("og") # OGP画像用
  end
end