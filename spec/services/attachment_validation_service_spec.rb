# typed: false
# frozen_string_literal: true

RSpec.describe AttachmentValidationService, type: :model do
  it "有効なファイルサイズと許可されたMIMEタイプのファイルが検証を通過すること" do
    blob = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new("test content"),
      filename: "test.jpg",
      content_type: "image/jpeg"
    )

    expect(AttachmentValidationService.valid?(blob)).to be(true)
  end

  it "ファイルサイズが上限を超える場合、検証が失敗すること" do
    large_content = "x" * (AttachmentValidationService::MAX_FILE_SIZE + 1)
    blob = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new(large_content),
      filename: "large.jpg",
      content_type: "image/jpeg"
    )

    service = AttachmentValidationService.new(blob)
    expect(service.valid?).to be(false)
    expect(service.errors).to include("ファイルサイズが大きすぎます（最大50MB）")
  end

  it "許可されていないMIMEタイプのファイルが検証に失敗すること" do
    blob = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new("#!/bin/bash\necho 'hello'"),
      filename: "script.sh",
      content_type: "application/x-sh"
    )

    service = AttachmentValidationService.new(blob)
    expect(service.valid?).to be(false)
    expect(service.errors).to include("許可されていないファイル形式です")
  end

  it "実際のファイル内容とMIMEタイプが一致しない場合、検証が失敗すること" do
    # HTMLコンテンツだが、MIMEタイプをJPEGと偽装
    html_content = "<html><body>test</body></html>"
    blob = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new(html_content),
      filename: "fake.jpg",
      content_type: "image/jpeg"
    )

    service = AttachmentValidationService.new(blob)
    expect(service.valid?).to be(false)
    expect(service.errors).to include("ファイルの内容が不正です")
  end
end