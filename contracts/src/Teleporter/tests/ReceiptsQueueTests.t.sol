// (c) 2023, Ava Labs, Inc. All rights reserved.
// See the file LICENSE for licensing terms.

// SPDX-License-Identifier: Ecosystem

pragma solidity 0.8.18;

import "forge-std/Test.sol";
import "../ReceiptQueue.sol";

contract ReceiptQueueTest is Test {
    using ReceiptQueue for ReceiptQueue.TeleporterMessageReceiptQueue;
    // The state of the contract gets reset before each
    // test is run, with the `setUp()` function being called
    // each time after deployment.
    ReceiptQueue.TeleporterMessageReceiptQueue public queue;

     // Add 3 elements to the queue.
    TeleporterMessageReceipt private receipt1 = TeleporterMessageReceipt({
        receivedMessageID: 543,
        relayerRewardAddress: 0x10eB43ef5982628728E3E4bb9F78834f67Fbb40b
    });
    TeleporterMessageReceipt private receipt2 = TeleporterMessageReceipt({
        receivedMessageID: 684384,
        relayerRewardAddress: 0x10eB43ef5982628728E3E4bb9F78834f67Fbb40b
    });
    TeleporterMessageReceipt private receipt3 = TeleporterMessageReceipt({
        receivedMessageID: 654351,
        relayerRewardAddress: 0xcC8E718045817AebA89592C72Ae1C9917f5D0894
    });

    function testReceiptQueue() public {
        // Check the initial size is zero.
        assertEq(queue.size(), 0);

        queue.enqueue(receipt1);
        queue.enqueue(receipt2);
        queue.enqueue(receipt3);

        // Check the size is now three.
        assertEq(queue.size(), 3);

        // Dequeue 3 elements and check they are given in the correct order (FIFO).
        TeleporterMessageReceipt memory result = queue.dequeue();
        assertEq(result.receivedMessageID, receipt1.receivedMessageID);
        assertEq(result.relayerRewardAddress, receipt1.relayerRewardAddress);
        result = queue.dequeue();
        assertEq(result.receivedMessageID, receipt2.receivedMessageID);
        assertEq(result.relayerRewardAddress, receipt2.relayerRewardAddress);
        result = queue.dequeue();
        assertEq(result.receivedMessageID, receipt3.receivedMessageID);
        assertEq(result.relayerRewardAddress, receipt3.relayerRewardAddress);

        // Check the size is now 0 again.
        assertEq(queue.size(), 0);

        // Enqueue two more of the same item to check you can have duplicates, followed by the second and third.
        queue.enqueue(receipt1);
        queue.enqueue(receipt1);
        queue.enqueue(receipt2);
        queue.enqueue(receipt3);

        // Check the size again.
        assertEq(queue.size(), 4);

        // Finally dequeue the elements and once again check they are returned in the correct order.
        result = queue.dequeue();
        assertEq(result.receivedMessageID, receipt1.receivedMessageID);
        assertEq(result.relayerRewardAddress, receipt1.relayerRewardAddress);
        result = queue.dequeue();
        assertEq(result.receivedMessageID, receipt1.receivedMessageID);
        assertEq(result.relayerRewardAddress, receipt1.relayerRewardAddress);
        result = queue.dequeue();
        assertEq(result.receivedMessageID, receipt2.receivedMessageID);
        assertEq(result.relayerRewardAddress, receipt2.relayerRewardAddress);
        result = queue.dequeue();
        assertEq(result.receivedMessageID, receipt3.receivedMessageID);
        assertEq(result.relayerRewardAddress, receipt3.relayerRewardAddress);
    }

    function testDequeueEmptyQueue() public {
        // Check that you can't dequeue from empty queue.
        vm.expectRevert(ReceiptQueue.EmptyQueue.selector);
        queue.dequeue();
    }

    function testGetReceiptAtIndex() public {
        queue.enqueue(receipt1);
        queue.enqueue(receipt1);
        queue.enqueue(receipt2);
        queue.enqueue(receipt3);

        // Check the size again.
        assertEq(queue.size(), 4);

        // Check that you can get receipts at specific indexes.
        TeleporterMessageReceipt memory result = queue.getReceiptAtIndex(2);
        assertEq(result.receivedMessageID, receipt2.receivedMessageID);
        assertEq(result.relayerRewardAddress, receipt2.relayerRewardAddress);

        // Check  can't get an out of index element.
        vm.expectRevert(ReceiptQueue.OutofIndex.selector);
        result = queue.getReceiptAtIndex(4);
    }
}
