# typed: false
# frozen_string_literal: true

RSpec.describe User, type: :model do
  describe "#viewable_notebooks" do
    context "リストが存在するとき" do
      let!(:space) { create(:space) }
      let!(:viewer) { create(:user, space:) }
      let!(:notebook_a) { create(:notebook, :public, space:, name: "リストA") }
      let!(:notebook_b) { create(:notebook, :private, space:, name: "リストB") }
      let!(:notebook_c) { create(:notebook, :private, space:, name: "リストC") }

      before do
        create(:notebook, :private, space:, name: "リストD")

        create(:notebook_membership, :admin, space:, notebook: notebook_b, member: viewer)
        create(:notebook_membership, :member, space:, notebook: notebook_c, member: viewer)
      end

      it "閲覧可能なリストを返すこと" do
        expect(viewer.viewable_notebooks).to contain_exactly(notebook_a, notebook_b, notebook_c)
      end
    end
  end

  describe "#last_note_modified_notebooks" do
    context "リストが存在するとき" do
      let!(:space) { create(:space) }
      let!(:viewer) { create(:user, space:) }
      let!(:notebook_a) { create(:notebook, space:, name: "リストA") }
      let!(:notebook_b) { create(:notebook, space:, name: "リストB") }
      let!(:notebook_c) { create(:notebook, space:, name: "リストC") }
      let!(:notebook_d) { create(:notebook, space:, name: "リストD") }

      before do
        create(:notebook, space:, name: "リストE")

        create(:notebook_membership, space:, notebook: notebook_a, member: viewer, joined_at: Time.zone.parse("2024-08-18 0:00:00"), last_note_modified_at: nil)
        create(:notebook_membership, space:, notebook: notebook_b, member: viewer, joined_at: Time.zone.parse("2024-08-18 1:00:00"), last_note_modified_at: Time.zone.parse("2024-08-19 0:00:00"))
        create(:notebook_membership, space:, notebook: notebook_c, member: viewer, joined_at: Time.zone.parse("2024-08-18 2:00:00"), last_note_modified_at: Time.zone.parse("2024-08-19 1:00:00"))
        create(:notebook_membership, space:, notebook: notebook_d, member: viewer, joined_at: Time.zone.parse("2024-08-18 3:00:00"), last_note_modified_at: nil)
      end

      it "記事が編集された順にリストが取得できること" do
        expect(
          viewer.last_note_modified_notebooks.pluck(:name)
        ).to eq(%w[リストC リストB リストD リストA])
      end
    end
  end

  describe "#last_modified_notes" do
    context "記事が存在するとき" do
      let!(:space) { create(:space) }
      let!(:user_a) { create(:user, space:) }
      let!(:user_b) { create(:user, space:) }
      let!(:notebook) { create(:notebook, space:) }
      let!(:note_a) { create(:note, space:, notebook:, author: user_a, title: "ノートA") }
      let!(:note_b) { create(:note, space:, notebook:, author: user_b, title: "ノートB") }
      let!(:note_c) { create(:note, space:, notebook:, author: user_b, title: "ノートC") }

      before do
        create(:note_editorship, space:, note: note_a, editor: user_a, last_note_modified_at: Time.zone.parse("2024-08-18 0:00:00"))
        create(:note_editorship, space:, note: note_b, editor: user_b, last_note_modified_at: Time.zone.parse("2024-08-18 1:00:00"))
        create(:note_editorship, space:, note: note_c, editor: user_b, last_note_modified_at: Time.zone.parse("2024-08-18 2:00:00"))
        create(:note_editorship, space:, note: note_c, editor: user_a, last_note_modified_at: Time.zone.parse("2024-08-18 3:00:00"))
      end

      it "最後に編集した記事から取得できること" do
        expect(user_a.last_modified_notes.pluck(:title)).to eq(%w[ノートC ノートA])
      end
    end
  end
end
