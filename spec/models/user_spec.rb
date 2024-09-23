# typed: false
# frozen_string_literal: true

RSpec.describe User, type: :model do
  describe "#viewable_lists" do
    context "リストが存在するとき" do
      let!(:space) { create(:space) }
      let!(:viewer) { create(:user, space:) }
      let!(:list_a) { create(:list, :public, space:, name: "リストA") }
      let!(:list_b) { create(:list, :private, space:, name: "リストB") }
      let!(:list_c) { create(:list, :private, space:, name: "リストC") }

      before do
        create(:list, :private, space:, name: "リストD")

        create(:list_membership, :admin, space:, list: list_b, member: viewer)
        create(:list_membership, :member, space:, list: list_c, member: viewer)
      end

      it "閲覧可能なリストを返すこと" do
        expect(viewer.viewable_lists).to contain_exactly(list_a, list_b, list_c)
      end
    end
  end

  describe "#last_note_modified_lists" do
    context "リストが存在するとき" do
      let!(:space) { create(:space) }
      let!(:viewer) { create(:user, space:) }
      let!(:list_a) { create(:list, space:, name: "リストA") }
      let!(:list_b) { create(:list, space:, name: "リストB") }
      let!(:list_c) { create(:list, space:, name: "リストC") }
      let!(:list_d) { create(:list, space:, name: "リストD") }

      before do
        create(:list, space:, name: "リストE")

        create(:list_membership, space:, list: list_a, member: viewer, joined_at: Time.zone.parse("2024-08-18 0:00:00"), last_note_modified_at: nil)
        create(:list_membership, space:, list: list_b, member: viewer, joined_at: Time.zone.parse("2024-08-18 1:00:00"), last_note_modified_at: Time.zone.parse("2024-08-19 0:00:00"))
        create(:list_membership, space:, list: list_c, member: viewer, joined_at: Time.zone.parse("2024-08-18 2:00:00"), last_note_modified_at: Time.zone.parse("2024-08-19 1:00:00"))
        create(:list_membership, space:, list: list_d, member: viewer, joined_at: Time.zone.parse("2024-08-18 3:00:00"), last_note_modified_at: nil)
      end

      it "記事が編集された順にリストが取得できること" do
        expect(
          viewer.last_note_modified_lists.pluck(:name)
        ).to eq(%w[リストC リストB リストD リストA])
      end
    end
  end

  describe "#last_modified_notes" do
    context "記事が存在するとき" do
      let!(:space) { create(:space) }
      let!(:user_a) { create(:user, space:) }
      let!(:user_b) { create(:user, space:) }
      let!(:list) { create(:list, space:) }
      let!(:note_a) { create(:note, space:, list:, author: user_a, title: "ノートA") }
      let!(:note_b) { create(:note, space:, list:, author: user_b, title: "ノートB") }
      let!(:note_c) { create(:note, space:, list:, author: user_b, title: "ノートC") }

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
