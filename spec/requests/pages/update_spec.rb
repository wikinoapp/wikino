# typed: false
# frozen_string_literal: true

RSpec.describe "PATCH /s/:space_identifier/pages/:page_number", type: :request do
  it "ログインしていないとき、ログインページにリダイレクトすること" do
    space = create(:space_record, :small)
    page = create(:page_record, space_record: space)

    patch "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/sign_in")
  end

  it "別のスペースに参加しているとき、404ページが表示されること" do
    user = create(:user_record, :with_password)

    space = create(:space_record, :small)
    page = create(:page_record, space_record: space)

    other_space = create(:space_record)
    create(:space_member_record, space_record: other_space, user_record: user)

    sign_in(user_record: user)

    patch "/s/#{space.identifier}/pages/#{page.number}"

    expect(response.status).to eq(404)
  end

  it "スペースに参加している & ページのトピックに参加していないとき、404ページが表示されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, space_record: space)
    page = create(:page_record, space_record: space, topic_record: topic, title: "A Page")

    sign_in(user_record: user)

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      pages_edit_form: {
        topic_number: topic.number,
        title: "Updated Title",
        body: "Updated Body"
      }
    })

    expect(response.status).to eq(404)

    # 404になったのでページは更新されていないはず
    expect(page.reload.title).to eq("A Page")
  end

  it "スペースに参加している & ページのトピックに参加している & 入力値が不正なとき、エラーメッセージを表示すること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)

    page = create(:page_record, space_record: space, topic_record: topic, title: "A Page")

    sign_in(user_record: user)

    expect(page.title).to eq("A Page")

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      pages_edit_form: {
        topic_number: topic.number,
        title: "", # タイトルが空
        body: "Updated Body"
      }
    })

    expect(response.status).to eq(422)
    expect(response.body).to include("タイトルを入力してください")

    # バリデーションエラーになったのでページは更新されていないはず
    expect(page.title).to eq("A Page")
  end

  it "スペースに参加している & ページのトピックに参加している & 入力値が正しいとき、ページが更新できること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)

    page = create(:page_record, space_record: space, topic_record: topic, title: "A Page")

    sign_in(user_record: user)

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      pages_edit_form: {
        topic_number: topic.number,
        title: "Updated Title",
        body: "Updated Body"
      }
    })

    expect(response.status).to eq(302)
    expect(response).to redirect_to("/s/#{space.identifier}/pages/#{page.number}")

    expect(page.reload.title).to eq("Updated Title")
  end

  it "ページに添付ファイルが含まれるとき、page_attachment_referencesレコードが作成されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)

    page = create(:page_record, space_record: space, topic_record: topic, title: "A Page")

    # 添付ファイルレコードを作成
    blob = ActiveStorage::Blob.create_and_upload!(
      io: File.open(Rails.root.join("spec/fixtures/files/test-image.png")),
      filename: "test-image.png",
      content_type: "image/png"
    )
    active_storage_attachment = ActiveStorage::Attachment.create!(
      name: "file",
      record: space,
      blob:
    )
    attachment1 = AttachmentRecord.create!(
      space_id: space.id,
      active_storage_attachment_record: active_storage_attachment,
      attached_space_member_id: space_member.id,
      attached_at: Time.current
    )

    blob2 = ActiveStorage::Blob.create_and_upload!(
      io: File.open(Rails.root.join("spec/fixtures/files/test-image-2.png")),
      filename: "test-image-2.png",
      content_type: "image/png"
    )
    active_storage_attachment2 = ActiveStorage::Attachment.create!(
      name: "file",
      record: space,
      blob: blob2
    )
    attachment2 = AttachmentRecord.create!(
      space_id: space.id,
      active_storage_attachment_record: active_storage_attachment2,
      attached_space_member_id: space_member.id,
      attached_at: Time.current
    )

    sign_in(user_record: user)

    # 添付ファイルを含むbodyで更新
    body_with_attachments = <<~HTML
      <p>This is a page with attachments</p>
      <img src="/attachments/#{attachment1.id}" alt="Test Image 1">
      <p>Another paragraph</p>
      <a href="/attachments/#{attachment2.id}">Download File</a>
    HTML

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      pages_edit_form: {
        topic_number: topic.number,
        title: "Page with Attachments",
        body: body_with_attachments
      }
    })

    expect(response.status).to eq(302)

    # page_attachment_referencesレコードが作成されていることを確認
    references = PageAttachmentReferenceRecord.where(page_id: page.id)
    expect(references.count).to eq(2)
    expect(references.pluck(:attachment_id)).to contain_exactly(attachment1.id, attachment2.id)
  end

  it "ページの添付ファイル参照を削除したとき、対応するpage_attachment_referencesレコードが削除されること" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)

    # 添付ファイルレコードを作成
    blob = ActiveStorage::Blob.create_and_upload!(
      io: File.open(Rails.root.join("spec/fixtures/files/test-image.png")),
      filename: "test-image.png",
      content_type: "image/png"
    )
    active_storage_attachment = ActiveStorage::Attachment.create!(
      name: "file",
      record: space,
      blob:
    )
    attachment1 = AttachmentRecord.create!(
      space_id: space.id,
      active_storage_attachment_record: active_storage_attachment,
      attached_space_member_id: space_member.id,
      attached_at: Time.current
    )

    blob2 = ActiveStorage::Blob.create_and_upload!(
      io: File.open(Rails.root.join("spec/fixtures/files/test-image-2.png")),
      filename: "test-image-2.png",
      content_type: "image/png"
    )
    active_storage_attachment2 = ActiveStorage::Attachment.create!(
      name: "file",
      record: space,
      blob: blob2
    )
    attachment2 = AttachmentRecord.create!(
      space_id: space.id,
      active_storage_attachment_record: active_storage_attachment2,
      attached_space_member_id: space_member.id,
      attached_at: Time.current
    )

    # 2つの添付ファイルを含むページを作成
    body_with_two_attachments = <<~HTML
      <img src="/attachments/#{attachment1.id}">
      <a href="/attachments/#{attachment2.id}">File</a>
    HTML

    page = create(:page_record, 
      space_record: space, 
      topic_record: topic, 
      title: "A Page",
      body: body_with_two_attachments
    )

    # 手動で参照レコードを作成（本来はサービスで作成されるが、テストのため直接作成）
    PageAttachmentReferenceRecord.create!(page_id: page.id, attachment_id: attachment1.id)
    PageAttachmentReferenceRecord.create!(page_id: page.id, attachment_id: attachment2.id)

    sign_in(user_record: user)

    # 1つの添付ファイルだけを残す
    body_with_one_attachment = <<~HTML
      <p>Updated content</p>
      <img src="/attachments/#{attachment1.id}">
    HTML

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      pages_edit_form: {
        topic_number: topic.number,
        title: "Updated Page",
        body: body_with_one_attachment
      }
    })

    expect(response.status).to eq(302)

    # 1つの参照だけが残っていることを確認
    references = PageAttachmentReferenceRecord.where(page_id: page.id)
    expect(references.count).to eq(1)
    expect(references.first.attachment_id).to eq(attachment1.id)
  end

  it "存在しない添付ファイルIDが含まれる場合、その参照は作成されないこと" do
    user = create(:user_record, :with_password)
    space = create(:space_record, :small)
    space_member = create(:space_member_record, space_record: space, user_record: user)

    topic = create(:topic_record, space_record: space)
    create(:topic_member_record, space_record: space, topic_record: topic, space_member_record: space_member)

    page = create(:page_record, space_record: space, topic_record: topic, title: "A Page")

    sign_in(user_record: user)

    # 存在しない添付ファイルIDを含むbody
    body_with_invalid_attachment = <<~HTML
      <p>This has an invalid attachment</p>
      <img src="/attachments/non-existent-id">
    HTML

    patch("/s/#{space.identifier}/pages/#{page.number}", params: {
      pages_edit_form: {
        topic_number: topic.number,
        title: "Page with Invalid Attachment",
        body: body_with_invalid_attachment
      }
    })

    expect(response.status).to eq(302)

    # 参照レコードが作成されていないことを確認
    references = PageAttachmentReferenceRecord.where(page_id: page.id)
    expect(references.count).to eq(0)
  end
end
