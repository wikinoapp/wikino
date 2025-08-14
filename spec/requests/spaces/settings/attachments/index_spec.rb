# typed: false
# frozen_string_literal: true

RSpec.describe "GET /s/:space_identifier/settings/attachments", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record)

    get "/s/#{space.identifier}/settings/attachments"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "ログインしている & スペースに参加していないとき、404を返すこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/settings/attachments"

    expect(response.status).to eq(404)
  end

  it "ログインしている & 別のスペースに参加しているとき、404を返すこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    other_space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: other_space)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/settings/attachments"

    expect(response.status).to eq(404)
  end

  it "ログインしている & スペースの管理者のとき、添付ファイル一覧が表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    create(:space_member_record, user_record: user, space_record: space, role: SpaceMemberRole::Owner.serialize)

    sign_in(user_record: user)

    get "/s/#{space.identifier}/settings/attachments"

    expect(response.status).to eq(200)
    expect(response.body).to include("添付ファイル")
  end

  it "ログインしている & スペースの管理者のとき、添付ファイルが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    space_member = create(:space_member_record, user_record: user, space_record: space, role: SpaceMemberRole::Owner.serialize)

    # ActiveStorageのアタッチメントとBlobを作成
    blob = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new("test content"),
      filename: "test.txt",
      content_type: "text/plain"
    )
    active_storage_attachment = ActiveStorage::Attachment.create!(
      id: SecureRandom.uuid,
      name: "file",
      record_type: "SpaceRecord",
      record_id: space.id,
      blob_id: blob.id
    )

    # AttachmentRecordを作成
    AttachmentRecord.create!(
      id: SecureRandom.uuid,
      space_id: space.id,
      active_storage_attachment_id: active_storage_attachment.id,
      attached_space_member_id: space_member.id,
      attached_at: Time.current,
      processing_status: AttachmentProcessingStatus::Completed.serialize
    )

    sign_in(user_record: user)

    get "/s/#{space.identifier}/settings/attachments"

    expect(response.status).to eq(200)
    expect(response.body).to include("test.txt")
  end

  it "検索クエリが指定されたとき、ファイル名で絞り込まれること" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    space_member = create(:space_member_record, user_record: user, space_record: space, role: SpaceMemberRole::Owner.serialize)

    # 複数のファイルを作成
    blob1 = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new("content 1"),
      filename: "document.pdf",
      content_type: "application/pdf"
    )
    attachment1 = ActiveStorage::Attachment.create!(
      id: SecureRandom.uuid,
      name: "file",
      record_type: "SpaceRecord",
      record_id: space.id,
      blob_id: blob1.id
    )
    AttachmentRecord.create!(
      id: SecureRandom.uuid,
      space_id: space.id,
      active_storage_attachment_id: attachment1.id,
      attached_space_member_id: space_member.id,
      attached_at: Time.current,
      processing_status: AttachmentProcessingStatus::Completed.serialize
    )

    blob2 = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new("content 2"),
      filename: "image.png",
      content_type: "image/png"
    )
    attachment2 = ActiveStorage::Attachment.create!(
      id: SecureRandom.uuid,
      name: "file",
      record_type: "SpaceRecord",
      record_id: space.id,
      blob_id: blob2.id
    )
    AttachmentRecord.create!(
      id: SecureRandom.uuid,
      space_id: space.id,
      active_storage_attachment_id: attachment2.id,
      attached_space_member_id: space_member.id,
      attached_at: Time.current,
      processing_status: AttachmentProcessingStatus::Completed.serialize
    )

    sign_in(user_record: user)

    get "/s/#{space.identifier}/settings/attachments", params: {q: "document"}

    expect(response.status).to eq(200)

    # 検索結果が正しくフィルタリングされているか確認
    body = response.body

    # document.pdfは含まれるべき
    expect(body).to include("document.pdf")

    # image.pngは含まれないべき（ただし、ファイル名として表示されていなければOK）
    # HTMLにimage.pngが含まれていても、それがファイル名として表示されていなければ問題ない
    # より正確なテストにする
    doc = Nokogiri::HTML(body)
    filenames = doc.css("p.font-semibold").map(&:text)

    expect(filenames).to include("document.pdf")
    expect(filenames).not_to include("image.png")
  end

  it "ファイルタイプが指定されたとき、コンテントタイプで絞り込まれること" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    space_member = create(:space_member_record, user_record: user, space_record: space, role: SpaceMemberRole::Owner.serialize)

    # 異なるタイプのファイルを作成
    blob1 = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new("content 1"),
      filename: "document.pdf",
      content_type: "application/pdf"
    )
    attachment1 = ActiveStorage::Attachment.create!(
      id: SecureRandom.uuid,
      name: "file",
      record_type: "SpaceRecord",
      record_id: space.id,
      blob_id: blob1.id
    )
    AttachmentRecord.create!(
      id: SecureRandom.uuid,
      space_id: space.id,
      active_storage_attachment_id: attachment1.id,
      attached_space_member_id: space_member.id,
      attached_at: Time.current,
      processing_status: AttachmentProcessingStatus::Completed.serialize
    )

    blob2 = ActiveStorage::Blob.create_and_upload!(
      io: StringIO.new("content 2"),
      filename: "image.png",
      content_type: "image/png"
    )
    attachment2 = ActiveStorage::Attachment.create!(
      id: SecureRandom.uuid,
      name: "file",
      record_type: "SpaceRecord",
      record_id: space.id,
      blob_id: blob2.id
    )
    AttachmentRecord.create!(
      id: SecureRandom.uuid,
      space_id: space.id,
      active_storage_attachment_id: attachment2.id,
      attached_space_member_id: space_member.id,
      attached_at: Time.current,
      processing_status: AttachmentProcessingStatus::Completed.serialize
    )

    sign_in(user_record: user)

    get "/s/#{space.identifier}/settings/attachments", params: {file_type: "image"}

    expect(response.status).to eq(200)
    expect(response.body).not_to include("document.pdf")
    expect(response.body).to include("image.png")
  end

  it "ページネーションが機能すること" do
    user = create(:user_record, :with_password)
    space = create(:space_record)
    space_member = create(:space_member_record, user_record: user, space_record: space, role: SpaceMemberRole::Owner.serialize)

    # 51個のファイルを作成（1ページ50件なので2ページ目が必要）
    51.times do |i|
      blob = ActiveStorage::Blob.create_and_upload!(
        io: StringIO.new("content #{i}"),
        filename: "file_#{i}.txt",
        content_type: "text/plain"
      )
      attachment = ActiveStorage::Attachment.create!(
        id: SecureRandom.uuid,
        name: "file",
        record_type: "SpaceRecord",
        record_id: space.id,
        blob_id: blob.id
      )
      AttachmentRecord.create!(
        id: SecureRandom.uuid,
        space_id: space.id,
        active_storage_attachment_id: attachment.id,
        attached_space_member_id: space_member.id,
        attached_at: Time.current - i.hours,
        processing_status: AttachmentProcessingStatus::Completed.serialize
      )
    end

    sign_in(user_record: user)

    # 1ページ目
    get "/s/#{space.identifier}/settings/attachments"
    expect(response.status).to eq(200)
    expect(response.body).to include("file_0.txt")
    expect(response.body).not_to include("file_50.txt")

    # 2ページ目
    get "/s/#{space.identifier}/settings/attachments", params: {page: 2}
    expect(response.status).to eq(200)
    expect(response.body).not_to include("file_0.txt")
    expect(response.body).to include("file_50.txt")
  end
end
